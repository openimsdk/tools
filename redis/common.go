package cache

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

// validateAddress checks the format of the given Redis address.
func validateAddress(address string) error {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return errs.WrapMsg(fmt.Errorf("invalid address format: %s", address), "invalid address format", "address", address)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil || port <= 0 || port > 65535 {
		return errs.WrapMsg(err, "invalid port in address", "address", address)
	}
	return nil
}
