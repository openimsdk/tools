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
	"github.com/OpenIMSDK/tools/errs"
	"github.com/go-zookeeper/zk"
)

func (s *ZkClient) RegisterConf2Registry(key string, conf []byte) error {
	path := s.getPath(key)

	exists, _, err := s.conn.Exists(path)
	if err != nil {
		return errs.Wrap(err, "checking existence for path %s in ZkClient RegisterConf2Registry", path)
	}

	if exists {
		if err := s.conn.Delete(path, 0); err != nil {
			return errs.Wrap(err, "deleting existing node for path %s in ZkClient RegisterConf2Registry", path)
		}
	}
	_, err = s.conn.Create(path, conf, 0, zk.WorldACL(zk.PermAll))
	if err != nil && err != zk.ErrNodeExists {
		return errs.Wrap(err, "creating node for path %s in ZkClient RegisterConf2Registry", path)
	}
	return nil
}

func (s *ZkClient) GetConfFromRegistry(key string) ([]byte, error) {
	path := s.getPath(key)
	bytes, _, err := s.conn.Get(path)
	if err != nil {
		return nil, errs.Wrap(err, "getting configuration for path %s from registry", path)
	}
	return bytes, nil
}
