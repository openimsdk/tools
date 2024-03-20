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

package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	ClusterMode       bool
	Address           []string
	Username          string
	Password          string
	EnablePipeline    bool
	MaxRetries        int
	DB                int
	PoolSize          int
	ConnectionTimeout time.Duration
}

func NewRedisClient(ctx context.Context, config *RedisConfig) (redis.UniversalClient, error) {
	if len(config.Address) == 0 {
		return nil, errors.New("redis address is empty")
	}

	var client redis.UniversalClient
	if len(config.Address) > 1 || config.ClusterMode {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:      config.Address,
			Username:   config.Username,
			Password:   config.Password,
			PoolSize:   config.PoolSize,
			MaxRetries: config.MaxRetries,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:       config.Address[0],
			Username:   config.Username,
			Password:   config.Password,
			DB:         config.DB,
			PoolSize:   config.PoolSize,
			MaxRetries: config.MaxRetries,
		})
	}

	cCtx, cancel := context.WithTimeout(ctx, config.ConnectionTimeout)
	defer cancel()

	if err := client.Ping(cCtx).Err(); err != nil {
		errMsg := fmt.Sprintf("Redis connection failed. Address: %v, Username: %s, ClusterMode: %t", config.Address, config.Username, config.ClusterMode)
		return nil, fmt.Errorf("%s, Error: %v", errMsg, err)
	}

	return client, nil
}
