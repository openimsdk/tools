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

package config

type MYSQL struct {
	Address       []string `yaml:"address"`
	Username      string   `yaml:"username"`
	Password      string   `yaml:"password"`
	Database      string   `yaml:"database"`
	MaxOpenConn   int      `yaml:"maxOpenConn"`
	MaxIdleConn   int      `yaml:"maxIdleConn"`
	MaxLifeTime   int      `yaml:"maxLifeTime"`
	LogLevel      int      `yaml:"logLevel"`
	SlowThreshold int      `yaml:"slowThreshold"`
}

type Zookeeper struct {
	Schema   string   `yaml:"schema"`
	ZkAddr   []string `yaml:"address"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

type Mongo struct {
	Uri         string   `yaml:"uri"`
	Address     []string `yaml:"address"`
	Database    string   `yaml:"database"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	MaxPoolSize int      `yaml:"maxPoolSize"`
}

type Redis struct {
	ClusterMode    bool     `yaml:"clusterMode"`
	Address        []string `yaml:"address"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	EnablePipeline bool     `yaml:"enablePipeline"`
}
type Kafka struct {
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	ProducerAck  string   `yaml:"producerAck"`
	CompressType string   `yaml:"compressType"`
	Addr         []string `yaml:"addr"`
	TLS          *struct {
		CACrt              string `yaml:"caCrt"`
		ClientCrt          string `yaml:"clientCrt"`
		ClientKey          string `yaml:"clientKey"`
		ClientKeyPwd       string `yaml:"clientKeyPwd"`
		InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	} `yaml:"tls"`
	LatestMsgToRedis struct {
		Topic string `yaml:"topic"`
	} `yaml:"latestMsgToRedis"`
	MsgToMongo struct {
		Topic string `yaml:"topic"`
	} `yaml:"offlineMsgToMongo"`
	MsgToPush struct {
		Topic string `yaml:"topic"`
	} `yaml:"msgToPush"`
	ConsumerGroupID struct {
		MsgToRedis string `yaml:"msgToRedis"`
		MsgToMongo string `yaml:"msgToMongo"`
		MsgToMySql string `yaml:"msgToMySql"`
		MsgToPush  string `yaml:"msgToPush"`
	} `yaml:"consumerGroupID"`
}

type Object struct {
	Enable string `yaml:"enable"`
	ApiURL string `yaml:"apiURL"`
	Minio  struct {
		Bucket          string `yaml:"bucket"`
		Endpoint        string `yaml:"endpoint"`
		AccessKeyID     string `yaml:"accessKeyID"`
		SecretAccessKey string `yaml:"secretAccessKey"`
		SessionToken    string `yaml:"sessionToken"`
		SignEndpoint    string `yaml:"signEndpoint"`
		PublicRead      bool   `yaml:"publicRead"`
	} `yaml:"minio"`
	Cos struct {
		BucketURL    string `yaml:"bucketURL"`
		SecretID     string `yaml:"secretID"`
		SecretKey    string `yaml:"secretKey"`
		SessionToken string `yaml:"sessionToken"`
		PublicRead   bool   `yaml:"publicRead"`
	} `yaml:"cos"`
	Oss struct {
		Endpoint        string `yaml:"endpoint"`
		Bucket          string `yaml:"bucket"`
		BucketURL       string `yaml:"bucketURL"`
		AccessKeyID     string `yaml:"accessKeyID"`
		AccessKeySecret string `yaml:"accessKeySecret"`
		SessionToken    string `yaml:"sessionToken"`
		PublicRead      bool   `yaml:"publicRead"`
	} `yaml:"oss"`
	Kodo struct {
		Endpoint        string `yaml:"endpoint"`
		Bucket          string `yaml:"bucket"`
		BucketURL       string `yaml:"bucketURL"`
		AccessKeyID     string `yaml:"accessKeyID"`
		AccessKeySecret string `yaml:"accessKeySecret"`
		SessionToken    string `yaml:"sessionToken"`
		PublicRead      bool   `yaml:"publicRead"`
	} `yaml:"kodo"`
}

type Mysql struct {
	Address       []string `yaml:"address"`
	Username      string   `yaml:"username"`
	Password      string   `yaml:"password"`
	Database      string   `yaml:"database"`
	MaxOpenConn   int      `yaml:"maxOpenConn"`
	MaxIdleConn   int      `yaml:"maxIdleConn"`
	MaxLifeTime   int      `yaml:"maxLifeTime"`
	LogLevel      int      `yaml:"logLevel"`
	SlowThreshold int      `yaml:"slowThreshold"`
}
