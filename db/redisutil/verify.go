package redisutil

import (
	"context"

	"github.com/openimsdk/tools/errs"
)

// CheckRedis checks the Redis connection.
func Check(ctx context.Context, config *Config) error {
	client, err := NewRedisClient(ctx, config)
	if err != nil {
		return err
	}
	defer client.Close()

	// Ping the Redis server to check connectivity.
	if err := client.Ping(ctx).Err(); err != nil {
		return errs.WrapMsg(err, "Redis ping failed", "config", config)
	}

	return nil
}
