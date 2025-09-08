package redistimer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openimsdk/tools/db/redisutil"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/timer"
	"github.com/redis/go-redis/v9"
)

// RedisTimer implements timer.Manager using Redis
type RedisTimer[T any] struct {
	client           redis.UniversalClient
	keyFunc          timer.KeyFunc[T]
	handlers         timer.HandlerMap[T]
	marshal          func(T) ([]byte, error)
	unmarshal        func([]byte) (T, error)
	namespace        string
	pollInterval     time.Duration
	batchSize        int
	selfContainedKey bool // true when key contains all data (string type only)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Redis-based timer manager and starts processing
func New[T any](ctx context.Context, redisConfig *redisutil.Config, keyFunc timer.KeyFunc[T], handlers timer.HandlerMap[T], opts ...Option[T]) (timer.Manager[T], error) {
	client, err := redisutil.NewRedisClient(ctx, redisConfig)
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to create redis client")
	}

	return NewWithClient[T](ctx, client, keyFunc, handlers, opts...)
}

// NewWithClient creates a timer manager with existing Redis client and starts processing
func NewWithClient[T any](ctx context.Context, client redis.UniversalClient, keyFunc timer.KeyFunc[T], handlers timer.HandlerMap[T], opts ...Option[T]) (timer.Manager[T], error) {
	if keyFunc == nil {
		return nil, errs.ErrArgs.WrapMsg("keyFunc is required")
	}
	if len(handlers) == 0 {
		return nil, errs.ErrArgs.WrapMsg("at least one handler is required")
	}

	ctx, cancel := context.WithCancel(ctx)
	rt := &RedisTimer[T]{
		client:       client,
		keyFunc:      keyFunc,
		handlers:     handlers,
		namespace:    "timer",
		pollInterval: 5 * time.Second,
		batchSize:    100,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Set default marshal/unmarshal using JSON
	rt.marshal = func(item T) ([]byte, error) {
		return json.Marshal(item)
	}
	rt.unmarshal = func(data []byte) (T, error) {
		var item T
		err := json.Unmarshal(data, &item)
		return item, err
	}

	// Apply options
	for _, opt := range opts {
		opt(rt)
	}

	// Start processing in a single goroutine
	rt.wg.Add(1)
	go rt.process()

	return rt, nil
}

// getKey returns the Redis key for timers of a specific type
func (r *RedisTimer[T]) getKey(timerType string) string {
	return fmt.Sprintf("%s:%s:timer", r.namespace, timerType)
}

// getDataKey returns the Redis key for storing item data of a specific type
func (r *RedisTimer[T]) getDataKey(timerType string) string {
	return fmt.Sprintf("%s:%s:data", r.namespace, timerType)
}

// process handles expired timers for all types
func (r *RedisTimer[T]) process() {
	defer r.wg.Done()
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			// Process each timer type
			for timerType, handler := range r.handlers {
				if err := r.processExpired(timerType, handler); err != nil {
					log.ZError(r.ctx, "failed to process expired timers", err, "type", timerType)
				}
			}
		}
	}
}

// processExpired processes expired items for a specific type
func (r *RedisTimer[T]) processExpired(timerType string, handler timer.Handler[T]) error {
	items, err := r.getExpired(timerType, r.batchSize)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	timerKey := r.getKey(timerType)
	dataKey := r.getDataKey(timerType)

	// Create pipeline for batch removal
	pipe := r.client.Pipeline()
	successCount := 0

	for _, item := range items {
		itemKey := r.keyFunc(item)

		// Execute handler
		if err = handler(r.ctx, item); err != nil {
			log.ZError(r.ctx, "handler failed, keeping timer", err, "timerType", timerType, "key", itemKey)
			continue
		}

		// Add to batch removal
		pipe.ZRem(r.ctx, timerKey, itemKey)

		// If not self-contained, also remove from hash
		if !r.selfContainedKey {
			pipe.HDel(r.ctx, dataKey, itemKey)
		}

		successCount++
	}

	// Execute batch removal if any items were successfully processed
	if successCount > 0 {
		if _, err := pipe.Exec(r.ctx); err != nil {
			return errs.WrapMsg(err, "failed to remove processed items", "timerType", timerType, "count", successCount)
		}
	}

	return nil
}

// Close releases resources and stops processing
func (r *RedisTimer[T]) Close() error {
	r.cancel()
	r.wg.Wait()
	return nil
}
