package redisutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRedisClient(t *testing.T) {
	config := &Config{
		ClusterMode: true,
		Address:     []string{"dev-im-scca8t.serverless.use2.cache.amazonaws.com:6379"},
		Username:    "default",
		Password:    "ToXjvU8tXHxtPmL8OLclWT6jVD",
		MaxRetry:    3,
		DB:          0,
		PoolSize:    10,
		TLSEnabled:  true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := NewRedisClient(ctx, config)
	assert.NoError(t, err)
	defer client.Close()

	err = client.Ping(ctx).Err()
	assert.NoError(t, err)
}
