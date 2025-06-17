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
	"crypto/tls"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/mw/specialerror"
	"github.com/openimsdk/tools/xtls"
	"github.com/redis/go-redis/v9"
)

func init() {
	if err := specialerror.AddReplace(redis.Nil, errs.ErrRecordNotFound); err != nil {
		panic(err)
	}
}

// RedisMode
const (
	ClusterMode    = "cluster"    // Cluster mode for Redis.
	SentinelMode   = "sentinel"   // Sentinel mode for Redis.
	StandaloneMode = "standalone" // Standalone mode for Redis.
)

// Config defines the configuration parameters for a Redis client, including
// options for both single-node and cluster mode connections.
type Config struct {
	RedisMode string             // RedisMode can be "cluster", "sentinel", or "standalone".
	Address   []string           // List of Redis server addresses (host:port).
	Username  string             // Username for Redis authentication (Redis 6 ACL).
	Password  string             // Password for Redis authentication.
	MaxRetry  int                // Maximum number of retries for a command.
	DB        int                // Database number to connect to, for non-cluster mode.
	PoolSize  int                // Number of connections to pool.
	TLS       *xtls.ClientConfig // TLS configuration for secure connections.
	Sentinel  *Sentinel          // Sentinel configuration for high availability.
}

type Sentinel struct {
	MasterName    string   `yaml:"masterName"`
	SentinelAddrs []string `yaml:"sentinelsAddrs"`
}

func NewRedisClient(ctx context.Context, config *Config) (redis.UniversalClient, error) {
	if len(config.Address) == 0 {
		return nil, errs.New("redis address is empty").Wrap()
	}

	if config.RedisMode == SentinelMode && config.Sentinel != nil {
		if config.Sentinel.MasterName == "" {
			return nil, errs.New("sentinel master name is required").Wrap()
		}
		if len(config.Sentinel.SentinelAddrs) == 0 {
			return nil, errs.New("sentinel addresses are required").Wrap()
		}
	}

	var tlsConf *tls.Config
	if config.TLS != nil {
		var err error
		tlsConf, err = config.TLS.ClientTLSConfig()
		if err != nil {
			return nil, errs.WrapMsg(err, "failed to get TLS config")
		}
	}
	var cli redis.UniversalClient
	if config.Sentinel != nil && config.RedisMode == SentinelMode {
		opt := &redis.FailoverOptions{
			MasterName:    config.Sentinel.MasterName,
			SentinelAddrs: config.Sentinel.SentinelAddrs,
			Username:      config.Username,
			Password:      config.Password,
			DB:            config.DB,
			PoolSize:      config.PoolSize,
			MaxRetries:    config.MaxRetry,
			RouteByLatency: true,
			RouteRandomly:  true,
			TLSConfig:     tlsConf,
		}
		cli = redis.NewFailoverClient(opt)
	} else if config.RedisMode == ClusterMode && len(config.Address) > 1 {
		opt := &redis.ClusterOptions{
			Addrs:      config.Address,
			Username:   config.Username,
			Password:   config.Password,
			PoolSize:   config.PoolSize,
			MaxRetries: config.MaxRetry,
			TLSConfig:  tlsConf,
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
			TLSConfig:  tlsConf,
		}
		cli = redis.NewClient(opt)
	}
	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, errs.WrapMsg(err, "Redis Ping failed", "Address", config.Address, "Username", config.Username, "RedisMode", config.RedisMode)
	}
	return cli, nil
}
