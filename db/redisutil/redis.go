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
	MasterName     string   `json:"masterName" yaml:"masterName" config:"allowempty"`
	SentinelAddrs  []string `json:"sentinelAddrs" yaml:"sentinelAddrs" config:"allowempty"`
	RouteByLatency bool     `json:"routeByLatency" yaml:"routeByLatency" config:"allowempty"` // Route by latency if true.
	RouteRandomly  bool     `json:"routeRandomly" yaml:"routeRandomly" config:"allowempty"`   // Route randomly if true.
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
			MasterName:     config.Sentinel.MasterName,
			SentinelAddrs:  config.Sentinel.SentinelAddrs,
			Username:       config.Username,
			Password:       config.Password,
			DB:             config.DB,
			PoolSize:       config.PoolSize,
			MaxRetries:     config.MaxRetry,
			RouteByLatency: config.Sentinel.RouteByLatency,
			RouteRandomly:  config.Sentinel.RouteRandomly,
			TLSConfig:      tlsConf,
		}
		if opt.RouteByLatency || opt.RouteRandomly {
			cli = redis.NewFailoverClusterClient(opt)
		} else {
			cli = redis.NewFailoverClient(opt)
		}
	} else if config.RedisMode == ClusterMode {
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
