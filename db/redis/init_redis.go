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

package redis

import (
	"context"
	"time"

	"github.com/openimsdk/tools/errs"
	"github.com/redis/go-redis/v9"
)

// RedisClientInterface defines the behavior of a Redis client. This interface
// can be implemented by any Redis client, facilitating testing and different
// implementations.
// RedisClient represents a Redis client, providing an interface for operations.
type Client interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

// RedisConfig defines the configuration parameters for a Redis client, including
// options for both single-node and cluster mode connections.
type Config struct {
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

func NewRedisClient(ctx context.Context, config *Config) (Client, error) {
	var err error
	once.Do(func() {
		if len(config.Address) == 0 {
			err = errs.New("redis address is empty").Wrap()
			return
		}

		clientOptions := getClientOptions(config)
		if config.ClusterMode || len(config.Address) > 1 {
			clientInstance = redis.NewClusterClient(clientOptions.(*redis.ClusterOptions))
		} else {
			clientInstance = redis.NewClient(clientOptions.(*redis.Options))
		}

		cCtx, cancel := context.WithTimeout(ctx, config.ConnectionTimeout)
		defer cancel()

		if pingErr := clientInstance.Ping(cCtx).Err(); pingErr != nil {
			err = errs.WrapMsg(pingErr, "Redis Ping failed", "Address", config.Address, "Username", config.Username, "ClusterMode", config.ClusterMode)
			return
		}
	})

	if err != nil {
		return nil, err
	}

	return clientInstance, nil
}
