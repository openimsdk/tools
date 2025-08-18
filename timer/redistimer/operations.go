package redistimer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/redis/go-redis/v9"
)

// Register adds an item with a timeout duration for a specific type
// If a timer with the same key already exists, it will be updated (upsert behavior)
func (r *RedisTimer[T]) Register(ctx context.Context, timerType string, item T, timeout time.Duration) error {
	return r.RegisterAt(ctx, timerType, item, time.Now().Add(timeout))
}

// RegisterAt adds an item that expires at a specific time for a specific type
func (r *RedisTimer[T]) RegisterAt(ctx context.Context, timerType string, item T, expireAt time.Time) error {
	itemKey := r.keyFunc(item)
	timerKey := r.getKey(timerType)
	score := float64(expireAt.Unix())

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Add to sorted set with expiration time as score
	pipe.ZAdd(ctx, timerKey, redis.Z{
		Score:  score,
		Member: itemKey,
	})

	// If not self-contained, also store data in hash
	if !r.selfContainedKey {
		data, err := r.marshal(item)
		if err != nil {
			return errs.WrapMsg(err, "failed to marshal item")
		}
		dataKey := r.getDataKey(timerType)
		pipe.HSet(ctx, dataKey, itemKey, data)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return errs.WrapMsg(err, "failed to register timer")
	}

	return nil
}

// Cancel removes a timer for an item of a specific type
func (r *RedisTimer[T]) Cancel(ctx context.Context, timerType string, key string) error {
	timerKey := r.getKey(timerType)

	pipe := r.client.Pipeline()
	pipe.ZRem(ctx, timerKey, key)

	// If not self-contained, also remove from hash
	if !r.selfContainedKey {
		dataKey := r.getDataKey(timerType)
		pipe.HDel(ctx, dataKey, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return errs.WrapMsg(err, "failed to cancel timer")
	}

	return nil
}

// getExpired retrieves items that have expired for a specific type
func (r *RedisTimer[T]) getExpired(timerType string, batchSize int) ([]T, error) {
	var result []T
	now := float64(time.Now().Unix())
	timerKey := r.getKey(timerType)

	// Get expired items from sorted set
	items, err := r.client.ZRangeByScore(r.ctx, timerKey, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%f", now),
		Count: int64(batchSize),
	}).Result()

	if err != nil {
		return nil, errs.WrapMsg(err, "failed to get expired items")
	}

	if len(items) == 0 {
		return result, nil
	}

	// If self-contained, keys are the data
	if r.selfContainedKey {
		// Convert keys directly to items (for string type)
		for _, itemKey := range items {
			// Type assertion: itemKey (string) -> T
			// This is safe because WithSelfContainedKey checks T is string
			if item, ok := any(itemKey).(T); ok {
				result = append(result, item)
			} else {
				log.ZError(r.ctx, "failed to convert key to item", nil, "key", itemKey)
			}
		}
		return result, nil
	}

	// Otherwise, fetch data from hash
	dataKey := r.getDataKey(timerType)

	// Get item data from hash using pipeline
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(items))
	for i, itemID := range items {
		cmds[i] = pipe.HGet(r.ctx, dataKey, itemID)
	}

	_, err = pipe.Exec(r.ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, errs.WrapMsg(err, "failed to get item data")
	}

	// Parse items
	for i, itemID := range items {
		data, err := cmds[i].Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				log.ZError(r.ctx, "item data not in hash", err, "itemID", itemID)
			}
			log.ZError(r.ctx, "failed to get item data", err, "itemID", itemID)
			continue
		}

		item, err := r.unmarshal([]byte(data))
		if err != nil {
			log.ZError(r.ctx, "failed to unmarshal item", err, "itemID", itemID)
			continue
		}

		result = append(result, item)
	}

	return result, nil
}

// GetPending returns the count of pending timers for a specific type
func (r *RedisTimer[T]) GetPending(ctx context.Context, timerType string) (int64, error) {
	key := r.getKey(timerType)
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, errs.WrapMsg(err, "failed to get pending count")
	}
	return count, nil
}
