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

package redis

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/openimsdk/tools/errs"
	"github.com/redis/go-redis/v9"
)

type clusterRedisClient struct {
	client *redis.ClusterClient
}

var (
	// Global singleton instance of the Redis client.
	clientInstance redis.UniversalClient
	once           sync.Once // Ensures client is initialized only once.
)

// getClientOptions returns the appropriate Redis options based on the configuration.
func getClientOptions(config *RedisConfig) interface{} {
	if config.ClusterMode || len(config.Address) > 1 {
		return &redis.ClusterOptions{
			Addrs:      config.Address,
			Username:   config.Username,
			Password:   config.Password,
			PoolSize:   config.PoolSize,
			MaxRetries: config.MaxRetries,
		}
	}
	return &redis.Options{
		Addr:       config.Address[0],
		Username:   config.Username,
		Password:   config.Password,
		DB:         config.DB,
		PoolSize:   config.PoolSize,
		MaxRetries: config.MaxRetries,
	}
}
