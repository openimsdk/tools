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
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/openimsdk/tools/errs"
)

type Config struct {
	ZkServers []string
	Scheme    string
	Username  string
	Password  string
	Timeout   time.Duration
}

func (s *ZkClient) RegisterConf2Registry(key string, conf []byte) error {
	path := s.getPath(key)

	exists, _, err := s.conn.Exists(path)
	if err != nil {
		return errs.WrapMsg(err, "Exists failed", "path", path)
	}

	if exists {
		if err := s.conn.Delete(path, 0); err != nil {
			return errs.WrapMsg(err, "Delete failed", "path", path)
		}
	}
	_, err = s.conn.Create(path, conf, 0, zk.WorldACL(zk.PermAll))
	if err != nil && err != zk.ErrNodeExists {
		return errs.WrapMsg(err, "Create failed", "path", path)
	}
	return nil
}

func (s *ZkClient) GetConfFromRegistry(key string) ([]byte, error) {
	path := s.getPath(key)
	bytes, _, err := s.conn.Get(path)
	if err != nil {
		return nil, errs.WrapMsg(err, "Get failed", "path", path)
	}
	return bytes, nil
}
