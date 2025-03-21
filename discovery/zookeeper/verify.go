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
	"context"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/openimsdk/tools/errs"
	"google.golang.org/grpc"
)

func Check(ctx context.Context, ZkServers []string, scheme string, options ...ZkOption) error {
	client := &ZkClient{
		ZkServers:  ZkServers,
		zkRoot:     "/",
		scheme:     scheme,
		timeout:    timeout,
		localConns: make(map[string][]*grpc.ClientConn),
		resolvers:  make(map[string]*Resolver),
		lock:       &sync.Mutex{},
		logger:     nilLog{},
	}
	for _, option := range options {
		option(client)
	}

	// Establish a Zookeeper connection with a specified timeout and handle authentication.
	conn, eventChan, err := zk.Connect(ZkServers, time.Duration(client.timeout)*time.Second)
	if err != nil {
		return errs.WrapMsg(err, "connect failed", "ZkServers", ZkServers)
	}

	_, cancel := context.WithCancel(context.Background())
	client.cancel = cancel
	client.ticker = time.NewTicker(defaultFreq)

	// Ensure authentication is set if credentials are provided.
	if client.username != "" && client.password != "" {
		auth := []byte(client.username + ":" + client.password)
		if err := conn.AddAuth("digest", auth); err != nil {
			conn.Close()
			return errs.WrapMsg(err, "AddAuth failed", "userName", client.username, "password", client.password)
		}
	}

	client.zkRoot += scheme
	client.eventChan = eventChan
	client.conn = conn

	// Verify root node existence and create if missing.
	if err := client.ensureRoot(); err != nil {
		conn.Close()
		return errs.WrapMsg(err, "ensureRoot failed", "zkRoot", client.zkRoot)
	}
	return nil
}
