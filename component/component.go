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
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	// defaultCfgPath is the default path of the configuration file.
	minioHealthCheckDuration = 1
	componentStartErrCode    = 6000
	configErrCode            = 6001
	mongoConnTimeout         = 30 * time.Second
	MaxRetry                 = 300
)

const (
	colorRed    = 31
	colorGreen  = 32
	colorYellow = 33
)

var (
	ErrComponentStart = errs.NewCodeError(componentStartErrCode, "ComponentStartErr")
	ErrConfig         = errs.NewCodeError(configErrCode, "Config file is incorrect")
)

// CheckMongo checks the MongoDB connection without retries
func CheckMongo(mongoStu *Mongo) (string, error) {
	mongodbHosts := strings.Join(mongoStu.Address, ",")
	if mongoStu.URL == "" {
		if mongoStu.Username != "" && mongoStu.Password != "" {
			mongoStu.URL = fmt.Sprintf("mongodb://%s:%s@%s/%s?maxPoolSize=%d",
				mongoStu.Username, mongoStu.Password, mongodbHosts, mongoStu.Database, mongoStu.MaxPoolSize)
		}
		mongoStu.URL = fmt.Sprintf("mongodb://%s/%s?maxPoolSize=%d",
			mongodbHosts, mongoStu.Database, mongoStu.MaxPoolSize)
	}

	ctx, cancel := context.WithTimeout(context.Background(), mongoConnTimeout)
	defer cancel()

	str := "ths uri is:" + strings.Join(mongoStu.Address, ",")

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoStu.URL))
	if err != nil {
		return "", errs.Wrap(ErrStr(err, str))
	}
	defer client.Disconnect(context.Background())

	ctx, cancel = context.WithTimeout(context.Background(), mongoConnTimeout)
	defer cancel()

	if err = client.Ping(ctx, nil); err != nil {
		return "", errs.Wrap(ErrStr(err, str))
	}

	return str, nil
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
func CheckMinio(minioStu *Minio) (string, error) {
	if minioStu.Endpoint == "" || minioStu.AccessKeyID == "" || minioStu.SecretAccessKey == "" {
		return "", ErrConfig.Wrap("MinIO configuration missing")
	}

	// Parse endpoint URL to determine if SSL is enabled
	u, err := url.Parse(minioStu.Endpoint)
	if err != nil {
		str := "the endpoint is:" + minioStu.Endpoint
		return "", errs.Wrap(ErrStr(err, str))
	}
	secure := u.Scheme == "https" || minioStu.UseSSL == "true"

	// Initialize MinIO client
	minioClient, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(minioStu.AccessKeyID, minioStu.SecretAccessKey, ""),
		Secure: secure,
	})
	str := "ths addr is:" + u.Host
	if err != nil {
		strs := fmt.Sprintf("%v;host:%s,accessKeyID:%s,secretAccessKey:%s,Secure:%v", err, u.Host, minioStu.AccessKeyID, minioStu.SecretAccessKey, secure)
		return "", errs.Wrap(err, strs)
	}

	// Perform health check
	cancel, err := minioClient.HealthCheck(time.Duration(minioHealthCheckDuration) * time.Second)
	if err != nil {
		return "", errs.Wrap(ErrStr(err, str))
	}
	defer cancel()

	if minioClient.IsOffline() {
		str := fmt.Sprintf("Minio server is offline;%s", str)
		return "", ErrComponentStart.Wrap(str)
	}

	// Check for localhost in API URL and Minio SignEndpoint
	if exactIP(minioStu.ApiURL) == "127.0.0.1" || exactIP(minioStu.SignEndpoint) == "127.0.0.1" {
		return "", ErrConfig.Wrap("apiURL or Minio SignEndpoint endpoint contain 127.0.0.1")
	}
	return str, nil
}

// CheckRedis checks the Redis connection
func CheckRedis(redisStu *Redis) (string, error) {
	// Split address to handle multiple addresses for cluster setup

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
	_, err := redisClient.Ping(context.Background()).Result()
	str := fmt.Sprintf("the addr is:%s", strings.Join(redisStu.Address, ","))
	if err != nil {
		strs := fmt.Sprintf("%s, the username is:%s, the password is:%s.", str, redisStu.Username, redisStu.Password)
		return "", errs.Wrap(ErrStr(err, strs))
	}

	return str, nil
}

// CheckZookeeper checks the Zookeeper connection
func CheckZookeeper(zkStu *Zookeeper) (string, error) {

	// Connect to Zookeeper
	str := fmt.Sprintf("the addr is:%s,the schema is:%s, the username is:%s, the password is:%s.", zkStu.ZkAddr, zkStu.Schema, zkStu.Username, zkStu.Password)
	c, eventChan, err := zk.Connect(zkStu.ZkAddr, time.Second) // Adjust the timeout as necessary
	if err != nil {
		return "", errs.Wrap(ErrStr(err, str))
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
			return "", errs.Wrap(ErrStr(errors.New("timeout waiting for Zookeeper connection"), str))
		}
	}
Connected:
	defer c.Close()

	return fmt.Sprintf("the address is:%s", zkStu.ZkAddr), nil
}

// CheckMySQL checks the mysql connection
func CheckMySQL(mysqlStu *MySQL) (string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		mysqlStu.Username,
		mysqlStu.Password,
		mysqlStu.Address[0],
		"mysql",
	)

	db, err := gorm.Open(mysql.Open(dsn), nil)
	if err != nil {
		return "", errs.Wrap(ErrStr(err, dsn))
	}
	sqlDB, err := db.DB()
	if err != nil {
		return "", errs.Wrap(err, "get sqlDB failed")
	}
	str := "the addr is:" + strings.Join(mysqlStu.Address, ",")
	defer sqlDB.Close()
	err = sqlDB.Ping()
	if err != nil {
		return "", errs.Wrap(err, "ping sqlDB failed")
	}

	return str, nil
}

// CheckKafka checks the Kafka connection
func CheckKafka(kafkaStu *Kafka) (string, sarama.Client, error) {
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
	str := "the addr is:" + strings.Join(kafkaStu.Addr, ",")
	kafkaClient, err := sarama.NewClient(kafkaStu.Addr, cfg)
	if err != nil {
		return "", nil, errs.Wrap(ErrStr(err, fmt.Sprintf("the address is:%s, the username is:%s, the password is:%s", kafkaStu.Addr, kafkaStu.Username, kafkaStu.Password)))
	}

	return str, kafkaClient, nil
}

func colorPrint(colorCode int, format string, a ...interface{}) {
	fmt.Printf("\x1b[%dm%s\x1b[0m\n", colorCode, fmt.Sprintf(format, a...))
}

func ErrorPrint(s string) {
	colorPrint(colorRed, "%v", s)
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
