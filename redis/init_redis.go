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
	"sync"
	"time"

	"github.com/openimsdk/tools/errs"
	"github.com/redis/go-redis/v9"
)

// RedisConfig defines the configuration parameters for a Redis client.
type RedisConfig struct {
	ClusterMode       bool          // Whether to use Redis in cluster mode.
	Address           []string      // List of Redis server addresses (host:port).
	Username          string        // Username for Redis authentication (Redis 6 ACL).
	Password          string        // Password for Redis authentication.
	EnablePipeline    bool          // Enable pipelining of commands for efficiency.
	MaxRetries        int           // Maximum number of retries for a command.
	DB                int           // Database number to connect to, for non-cluster mode.
	PoolSize          int           // Number of connections to pool.
	ConnectionTimeout time.Duration // Timeout for connecting to Redis servers.
}

// Global variables
var (
	redisClient redis.UniversalClient // Singleton instance of the Redis client.
	once        sync.Once             // Ensures the client is initialized only once.
)

// NewRedisClient creates a new Redis client.
func NewRedisClient(ctx context.Context, config *RedisConfig) (redis.UniversalClient, error) {
	var initErr error

	// Use sync.Once to ensure that the client initialization logic runs only once.
	once.Do(func() {
		if len(config.Address) == 0 {
			initErr = errs.Wrap(errors.New("redis address is empty"))
			return
		}

		if len(config.Address) > 1 || config.ClusterMode {
			redisClient = redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:      config.Address,
				Username:   config.Username,
				Password:   config.Password,
				PoolSize:   config.PoolSize,
				MaxRetries: config.MaxRetries,
			})
		} else {
			redisClient = redis.NewClient(&redis.Options{
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

		if err := redisClient.Ping(cCtx).Err(); err != nil {
			initErr = errs.WrapMsg(err, "Redis Ping failed.", "Address", "Address", config.Address, "Username", config.Username, "ClusterMode", config.ClusterMode)
			return
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return redisClient, nil
}
