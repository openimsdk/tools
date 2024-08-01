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

package redisutil

import (
	"context"

	"github.com/openimsdk/tools/errs"
	"github.com/redis/go-redis/v9"
)

// Config defines the configuration parameters for a Redis client, including
// options for both single-node and cluster mode connections.
type Config struct {
	ClusterMode bool     // Whether to use Redis in cluster mode.
	Address     []string // List of Redis server addresses (host:port).
	Username    string   // Username for Redis authentication (Redis 6 ACL).
	Password    string   // Password for Redis authentication.
	MaxRetry    int      // Maximum number of retries for a command.
	DB          int      // Database number to connect to, for non-cluster mode.
	PoolSize    int      // Number of connections to pool.
}

func NewRedisClient(ctx context.Context, config *Config) (redis.UniversalClient, error) {
	if len(config.Address) == 0 {
		return nil, errs.New("redis address is empty").Wrap()
	}
	var cli redis.UniversalClient
	if config.ClusterMode || len(config.Address) > 1 {
		opt := &redis.ClusterOptions{
			Addrs:      config.Address,
			Username:   config.Username,
			Password:   config.Password,
			PoolSize:   config.PoolSize,
			MaxRetries: config.MaxRetry,
		}
		cli = redis.NewClusterClient(opt)
	} else {
		opt := &redis.Options{
			Addr:       config.Address[0],
			Username:   config.Username,
			Password:   config.Password,
			DB:         config.DB,
			PoolSize:   config.PoolSize,
			MaxRetries: config.MaxRetry,
		}
		cli = redis.NewClient(opt)
	}
	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, errs.WrapMsg(err, "Redis Ping failed", "Address", config.Address, "Username", config.Username, "ClusterMode", config.ClusterMode)
	}
	return cli, nil
}
