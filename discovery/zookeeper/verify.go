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

package zookeeper

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/openimsdk/tools/errs"
)

// CheckZookeeper validates the connection to Zookeeper within a specific timeout.
func CheckZookeeper(zkStu *ZkClient) error {
	zkStuInfo, err := json.Marshal(zkStu)
	if err != nil {
		return errs.WrapMsg(err, "failed to marshal Zookeeper config", "config", zkStu)
	}

	// Initialize a Zookeeper connection with a specified timeout.
	conn, _, err := zk.Connect(zkStu.ZkServers, 5*time.Second)
	if err != nil {
		return errs.WrapMsg(err, "failed to connect to Zookeeper", "config", zkStu)
	}
	defer conn.Close()

	// Check for a successful session establishment.
	if conn.State() != zk.StateHasSession {
		return fmt.Errorf("failed to establish a session with Zookeeper within the timeout, config: %s", zkStuInfo)
	}

	return nil
}
