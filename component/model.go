// Copyright © 2024 OpenIM open source community. All rights reserved.
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

type Minio struct {
	ApiURL          string `json:"apiURL"`
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
	SignEndpoint    string `json:"signEndpoint"`
	UseSSL          string `json:"useSSL"`
}

type Zookeeper struct {
	Schema   string   `json:"schema"`
	ZkAddr   []string `json:"zkAddr"`
	Username string   `json:"username"`
	Password string   `json:"password"`
}

type Kafka struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Addr     []string `json:"addr"`
}
