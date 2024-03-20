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
	"sync"
	"time"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/redis/go-redis/v9"
)

var (
	once        sync.Once
	redisClient redis.UniversalClient
)

type RedisConfig struct {
	ClusterMode    bool
	Address        []string
	Username       string
	Password       string
	EnablePipeline bool
	MaxRetries     int
}

func NewRedisClient(ctx context.Context, config *RedisConfig) (redis.UniversalClient, error) {
	var initErr error

	once.Do(func() {
		if len(config.Address) == 0 {
			initErr = errors.New("redis address is empty")
			return
		}

		var client redis.UniversalClient
		if len(config.Address) > 1 || config.ClusterMode {
			client = redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:      config.Address,
				Username:   config.Username,
				Password:   config.Password,
				PoolSize:   50,
				MaxRetries: config.MaxRetries,
			})
		} else {
			client = redis.NewClient(&redis.Options{
				Addr:       config.Address[0],
				Username:   config.Username,
				Password:   config.Password,
				DB:         0,
				PoolSize:   100,
				MaxRetries: config.MaxRetries,
			})
		}
		cCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := client.Ping(cCtx).Err(); err != nil {
			errMsg := fmt.Sprintf("Redis connection failed. Address: %v, Username: %s, ClusterMode: %t", config.Address, config.Username, config.ClusterMode)
			initErr = fmt.Errorf("%s, Error: %v", errMsg, err)
			return
		}

		redisClient = client
		log.CInfo(ctx, "Redis connected successfully")

	})

	if initErr != nil {
		return nil, errs.Wrap(initErr)
	}

	return redisClient, nil
}
