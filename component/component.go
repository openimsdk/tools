// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/go-zookeeper/zk"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	// defaultCfgPath is the default path of the configuration file.
	minioHealthCheckDuration = 1
	mongoConnTimeout         = 30 * time.Second
	MaxRetry                 = 300
)

const (
	colorRed    = 31
	colorGreen  = 32
	colorYellow = 33
)

// CheckMongo checks the MongoDB connection without retries
func CheckMongo(mongoStu *Mongo) error {
	mongodbHosts := strings.Join(mongoStu.Address, ",")
	if mongoStu.URL == "" {
		if mongoStu.Username != "" && mongoStu.Password != "" {
			mongoStu.URL = fmt.Sprintf("mongodb://%s:%s@%s/%s?maxPoolSize=%d",
				mongoStu.Username, mongoStu.Password, mongodbHosts, mongoStu.Database, mongoStu.MaxPoolSize)
		}
		mongoStu.URL = fmt.Sprintf("mongodb://%s/%s?maxPoolSize=%d",
			mongodbHosts, mongoStu.Database, mongoStu.MaxPoolSize)
	}

	mogoInfo, err := json.Marshal(mongoStu)
	if err != nil {
		return errs.Wrap(errors.New("mongoStu Marshal failed"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), mongoConnTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoStu.URL))
	if err != nil {
		return errs.Wrap(ErrStr(err, string(mogoInfo)))
	}
	defer client.Disconnect(context.Background())

	ctx, cancel = context.WithTimeout(context.Background(), mongoConnTimeout)
	defer cancel()

	if err = client.Ping(ctx, nil); err != nil {
		return errs.Wrap(ErrStr(err, string(mogoInfo)))
	}
	return nil
}

func exactIP(urll string) string {
	u, _ := url.Parse(urll)
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}
	if strings.HasSuffix(host, ":") {
		host = host[0 : len(host)-1]
	}

	return host
}

// CheckMinio checks the MinIO connection
func CheckMinio(minioStu *Minio) error {
	if minioStu.Endpoint == "" || minioStu.AccessKeyID == "" || minioStu.SecretAccessKey == "" {
		return errs.Wrap(errors.New("MinIO configuration missing"))
	}

	// Parse endpoint URL to determine if SSL is enabled
	minioInfo, err := json.Marshal(minioStu)
	if err != nil {
		return errs.Wrap(errors.New("minioStu Marshal failed"))
	}
	u, err := url.Parse(minioStu.Endpoint)
	if err != nil {
		return errs.Wrap(ErrStr(err, string(minioInfo)))
	}
	secure := u.Scheme == "https" || minioStu.UseSSL == "true"

	// Initialize MinIO client
	minioClient, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(minioStu.AccessKeyID, minioStu.SecretAccessKey, ""),
		Secure: secure,
	})
	if err != nil {
		return errs.Wrap(ErrStr(err, string(minioInfo)))
	}

	// Perform health check
	cancel, err := minioClient.HealthCheck(time.Duration(minioHealthCheckDuration) * time.Second)
	if err != nil {
		return errs.Wrap(ErrStr(err, string(minioInfo)))
	}
	defer cancel()

	if minioClient.IsOffline() {
		return errs.Wrap(ErrStr(err, string(minioInfo)))
	}

	// Check for localhost in API URL and Minio SignEndpoint
	if exactIP(minioStu.ApiURL) == "127.0.0.1" || exactIP(minioStu.SignEndpoint) == "127.0.0.1" {
		return errs.Wrap(errors.New("apiURL or Minio SignEndpoint endpoint contain 127.0.0.1"), string(minioInfo))
	}
	return nil
}

// CheckRedis checks the Redis connection
func CheckRedis(redisStu *Redis) error {
	// Split address to handle multiple addresses for cluster setup

	redisInfo, err := json.Marshal(redisStu)
	if err != nil {
		return errs.Wrap(errors.New("redisStu Marshal failed"))
	}

	var redisClient redis.UniversalClient
	if len(redisStu.Address) > 1 {
		// Use cluster client for multiple addresses
		redisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    redisStu.Address,
			Username: redisStu.Username,
			Password: redisStu.Password,
		})
	} else {
		// Use regular client for single address
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisStu.Address[0],
			Username: redisStu.Username,
			Password: redisStu.Password,
		})
	}
	defer redisClient.Close()

	// Ping Redis to check connectivity
	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		return errs.Wrap(ErrStr(err, string(redisInfo)))
	}
	return nil
}

// CheckZookeeper checks the Zookeeper connection
func CheckZookeeper(zkStu *Zookeeper) error {

	zkStuInfo, err := json.Marshal(zkStu)
	if err != nil {
		return errs.Wrap(errors.New("redisStu Marshal failed"))
	}

	// Connect to Zookeeper
	c, eventChan, err := zk.Connect(zkStu.ZkAddr, time.Second) // Adjust the timeout as necessary
	if err != nil {
		return errs.Wrap(ErrStr(err, string(zkStuInfo)))
	}
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event := <-eventChan:
			if event.State == zk.StateConnected {
				fmt.Println("Connected to Zookeeper")
				goto Connected
			}
		case <-timeout:
			return errs.Wrap(ErrStr(errors.New("timeout waiting for Zookeeper connection"), string(zkStuInfo)))
		}
	}
Connected:
	defer c.Close()

	return nil
}

// CheckMySQL checks the mysql connection
func CheckMySQL(mysqlStu *MySQL) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		mysqlStu.Username,
		mysqlStu.Password,
		mysqlStu.Address[0],
		"mysql",
	)

	zkStuInfo, err := json.Marshal(mysqlStu)
	if err != nil {
		return errs.Wrap(errors.New("mysqlStu Marshal failed"))
	}

	db, err := gorm.Open(mysql.Open(dsn), nil)
	if err != nil {
		return errs.Wrap(ErrStr(err, dsn))
	}
	sqlDB, err := db.DB()
	if err != nil {
		return errs.Wrap(ErrStr(err, string(zkStuInfo)))
	}
	defer sqlDB.Close()
	err = sqlDB.Ping()
	if err != nil {
		return errs.Wrap(ErrStr(err, string(zkStuInfo)))
	}

	return nil
}

// CheckKafka checks the Kafka connection
func CheckKafka(kafkaStu *Kafka) (sarama.Client, error) {
	// Configure Kafka client
	cfg := sarama.NewConfig()
	if kafkaStu.Username != "" && kafkaStu.Password != "" {
		cfg.Net.SASL.Enable = true
		cfg.Net.SASL.User = kafkaStu.Username
		cfg.Net.SASL.Password = kafkaStu.Password
	}
	// Additional Kafka setup (e.g., TLS configuration) can be added here
	// kafka.SetupTLSConfig(cfg)

	// Create Kafka client
	kafkaInfo, err := json.Marshal(kafkaStu)
	if err != nil {
		return nil, errs.Wrap(errors.New("minioStu Marshal failed"))
	}
	kafkaClient, err := sarama.NewClient(kafkaStu.Addr, cfg)
	if err != nil {
		return nil, errs.Wrap(ErrStr(err, string(kafkaInfo)))
	}

	return kafkaClient, nil
}

func colorPrint(colorCode int, format string, a ...interface{}) {
	fmt.Printf("\x1b[%dm%s\x1b[0m\n", colorCode, fmt.Sprintf(format, a...))
}

func colorErrPrint(colorCode int, format string, a ...interface{}) {
	log.Printf("\x1b[%dm%s\x1b[0m\n", colorCode, fmt.Sprintf(format, a...))
}

func ErrorPrint(s string) {
	colorErrPrint(colorRed, "%v", s)
}

func SuccessPrint(s string) {
	colorPrint(colorGreen, "%v", s)
}

func WarningPrint(s string) {
	colorPrint(colorYellow, "Warning: But %v", s)
}

func ErrStr(err error, str string) error {
	return fmt.Errorf("%v;%s", err, str)
}
