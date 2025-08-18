// Copyright © 2025 OpenIM open source community. All rights reserved.
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

package redistask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openimsdk/tools/db/redisutil"
	"github.com/openimsdk/tools/queue/task"
	"github.com/redis/go-redis/v9"
)

type QueueManager[T any, K comparable] struct {
	client        redis.UniversalClient
	namespace     string
	maxProcessing int
	maxWaiting    int
	maxGlobal     int

	equalDataFunc        func(a, b T) bool
	afterProcessPushFunc []func(key K, data T)
	assignStrategy       func(ctx context.Context, m *QueueManager[T, K]) (K, bool)

	marshalFunc   func(T) ([]byte, error)
	unmarshalFunc func([]byte, *T) error
}

func NewQueueManager[T any, K comparable](
	ctx context.Context,
	config *redisutil.Config,
	maxGlobal, maxProcessing, maxWaiting int,
	equalFunc func(a, b T) bool,
	opts ...Option[T, K],
) (task.QueueManager[T, K], error) {
	client, err := redisutil.NewRedisClient(ctx, config)
	if err != nil {
		return nil, err
	}

	m := &QueueManager[T, K]{
		client:         client,
		namespace:      "default",
		maxProcessing:  maxProcessing,
		maxWaiting:     maxWaiting,
		maxGlobal:      maxGlobal,
		equalDataFunc:  equalFunc,
		assignStrategy: getStrategy[T, K](RoundRobin), // Default to round-robin
		marshalFunc: func(v T) ([]byte, error) {
			return json.Marshal(v)
		},
		unmarshalFunc: func(b []byte, v *T) error {
			return json.Unmarshal(b, v)
		},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

func (m *QueueManager[T, K]) getGlobalQueueKey() string {
	return fmt.Sprintf("taskqueue:%s:global", m.namespace)
}

func (m *QueueManager[T, K]) getKeysListKey() string {
	return fmt.Sprintf("taskqueue:%s:keys", m.namespace)
}

func (m *QueueManager[T, K]) getLastIndexKey() string {
	return fmt.Sprintf("taskqueue:%s:lastindex", m.namespace)
}

func (m *QueueManager[T, K]) getProcessingQueueKey(key K) string {
	return fmt.Sprintf("taskqueue:%s:processing:%v", m.namespace, key)
}

func (m *QueueManager[T, K]) getWaitingQueueKey(key K) string {
	return fmt.Sprintf("taskqueue:%s:waiting:%v", m.namespace, key)
}

func (m *QueueManager[T, K]) AddKey(ctx context.Context, key K) error {
	// Use Lua script to check if key's queue exists and add if not
	script := redis.NewScript(`
		local list_key = KEYS[1]
		local processing_queue_key = KEYS[2]
		local key_value = ARGV[1]
		
		-- Check if processing queue exists (indicating key already added)
		local exists = redis.call('EXISTS', processing_queue_key)
		if exists == 1 then
			return 0  -- Key already exists
		end
		
		-- Add key to the end of list
		redis.call('RPUSH', list_key, key_value)
		return 1  -- Key added
	`)

	keyStr := fmt.Sprintf("%v", key)
	_, err := script.Run(ctx, m.client,
		[]string{m.getKeysListKey(), m.getProcessingQueueKey(key)},
		keyStr).Result()
	return err
}

func (m *QueueManager[T, K]) Insert(ctx context.Context, data T) error {
	key, hasKey := m.assignStrategy(ctx, m)
	if !hasKey {
		// No key available, push to global queue
		return m.pushToGlobalQueue(ctx, data)
	}

	// Try to push to processing queue first
	if err := m.pushToProcessingQueue(ctx, key, data); err == nil {
		return nil
	}

	// If processing queue is full, push to global queue
	return m.pushToGlobalQueue(ctx, data)
}

func (m *QueueManager[T, K]) InsertByKey(ctx context.Context, key K, data T) error {
	// Ensure key exists
	if err := m.AddKey(ctx, key); err != nil {
		return err
	}

	// Try processing queue first
	if err := m.pushToProcessingQueue(ctx, key, data); err == nil {
		return nil
	}

	// Try waiting queue
	if err := m.pushToWaitingQueue(ctx, key, data); err == nil {
		return nil
	}

	return task.ErrWaitingQueueFull
}

func (m *QueueManager[T, K]) Delete(ctx context.Context, key K, data T) error {
	// Try to remove from processing queue first
	removed, err := m.removeFromQueue(ctx, m.getProcessingQueueKey(key), data)
	if err != nil {
		return err
	}
	if removed {
		// Backfill from waiting queue or global queue
		if err := m.backfillProcessingQueue(ctx, key); err != nil {
			// Log error but don't fail the delete operation
			// The delete was successful, backfill is best-effort
		}
		return nil
	}

	// Try to remove from waiting queue
	removed, err = m.removeFromQueue(ctx, m.getWaitingQueueKey(key), data)
	if err != nil {
		return err
	}
	if removed {
		return nil
	}

	return task.ErrDataNotFound
}

func (m *QueueManager[T, K]) DeleteKey(ctx context.Context, key K) error {
	keyStr := fmt.Sprintf("%v", key)

	// Remove key from the keys list (0 means remove all occurrences)
	if err := m.client.LRem(ctx, m.getKeysListKey(), 0, keyStr).Err(); err != nil {
		return err
	}

	// Clear queues
	return m.client.Del(ctx,
		m.getProcessingQueueKey(key),
		m.getWaitingQueueKey(key)).Err()
}

func (m *QueueManager[T, K]) GetProcessingQueueLengths(ctx context.Context) (map[K]int, error) {
	lengths := make(map[K]int)

	// Get all keys from Redis list
	keyStrs, err := m.client.LRange(ctx, m.getKeysListKey(), 0, -1).Result()
	if err != nil {
		return nil, err
	}

	// Get length for each key
	for _, keyStr := range keyStrs {
		var key K
		if _, err := fmt.Sscanf(keyStr, "%v", &key); err == nil {
			length, err := m.client.LLen(ctx, m.getProcessingQueueKey(key)).Result()
			if err != nil {
				return nil, err
			}
			lengths[key] = int(length)
		}
	}

	return lengths, nil
}

func (m *QueueManager[T, K]) TransformProcessingData(ctx context.Context, fromKey, toKey K, data T) error {
	// Remove from source processing queue
	removed, err := m.removeFromQueue(ctx, m.getProcessingQueueKey(fromKey), data)
	if err != nil {
		return err
	}
	if !removed {
		return task.ErrDataNotFound
	}

	// Try to add to target processing queue
	if err := m.pushToProcessingQueue(ctx, toKey, data); err != nil {
		// If target is full, add to waiting queue
		if err := m.pushToWaitingQueue(ctx, toKey, data); err != nil {
			// If both queues are full, push back to source to avoid data loss
			m.pushToProcessingQueue(ctx, fromKey, data)
			return err
		}
	}

	// Backfill source processing queue
	return m.backfillProcessingQueue(ctx, fromKey)
}

func (m *QueueManager[T, K]) pushToProcessingQueue(ctx context.Context, key K, data T) error {
	queueKey := m.getProcessingQueueKey(key)

	// Check current length
	length, err := m.client.LLen(ctx, queueKey).Result()
	if err != nil {
		return err
	}

	if int(length) >= m.maxProcessing {
		return task.ErrProcessingQueueFull
	}

	// Push to queue
	dataBytes, err := m.marshalFunc(data)
	if err != nil {
		return err
	}

	err = m.client.LPush(ctx, queueKey, dataBytes).Err()
	if err != nil {
		return err
	}

	// Call after process push functions
	for _, fn := range m.afterProcessPushFunc {
		fn(key, data)
	}

	return nil
}

func (m *QueueManager[T, K]) pushToWaitingQueue(ctx context.Context, key K, data T) error {
	queueKey := m.getWaitingQueueKey(key)

	// Check current length
	length, err := m.client.LLen(ctx, queueKey).Result()
	if err != nil {
		return err
	}

	if int(length) >= m.maxWaiting {
		return task.ErrWaitingQueueFull
	}

	// Push to queue
	dataBytes, err := m.marshalFunc(data)
	if err != nil {
		return err
	}

	return m.client.LPush(ctx, queueKey, dataBytes).Err()
}

func (m *QueueManager[T, K]) pushToGlobalQueue(ctx context.Context, data T) error {
	queueKey := m.getGlobalQueueKey()

	// Check current length
	length, err := m.client.LLen(ctx, queueKey).Result()
	if err != nil {
		return err
	}

	if int(length) >= m.maxGlobal {
		return task.ErrGlobalQueueFull
	}

	// Push to queue
	dataBytes, err := m.marshalFunc(data)
	if err != nil {
		return err
	}

	return m.client.LPush(ctx, queueKey, dataBytes).Err()
}

func (m *QueueManager[T, K]) removeFromQueue(ctx context.Context, queueKey string, data T) (bool, error) {
	// Get all items from queue
	items, err := m.client.LRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return false, err
	}

	dataBytes, err := m.marshalFunc(data)
	if err != nil {
		return false, err
	}
	targetStr := string(dataBytes)

	// Find and remove the item
	for i, item := range items {
		if item == targetStr {
			// Remove by index using Lua script for atomicity
			script := redis.NewScript(`
				local key = KEYS[1]
				local index = tonumber(ARGV[1])
				local value = redis.call('lindex', key, index)
				if value then
					redis.call('lset', key, index, '__REDIS_QUEUE_TOMBSTONE__')
					redis.call('lrem', key, 1, '__REDIS_QUEUE_TOMBSTONE__')
					return 1
				end
				return 0
			`)

			removed, err := script.Run(ctx, m.client, []string{queueKey}, i).Int()
			if err != nil {
				return false, err
			}

			return removed == 1, nil
		}
	}

	return false, nil
}

func (m *QueueManager[T, K]) backfillProcessingQueue(ctx context.Context, key K) error {
	processingKey := m.getProcessingQueueKey(key)
	waitingKey := m.getWaitingQueueKey(key)

	// Check if processing queue has space
	length, err := m.client.LLen(ctx, processingKey).Result()
	if err != nil {
		return err
	}
	if int(length) >= m.maxProcessing {
		return nil
	}

	// Try to pop from waiting queue first
	dataStr, err := m.client.RPop(ctx, waitingKey).Result()
	if err == nil {
		// Push to processing queue
		if err := m.client.LPush(ctx, processingKey, dataStr).Err(); err != nil {
			// Push back to waiting queue if failed
			m.client.RPush(ctx, waitingKey, dataStr)
			return err
		}

		// Decode and call after process push functions
		var data T
		if err := m.unmarshalFunc([]byte(dataStr), &data); err == nil {
			for _, fn := range m.afterProcessPushFunc {
				fn(key, data)
			}
		}
		return nil
	}

	// Try to pop from global queue
	globalKey := m.getGlobalQueueKey()
	dataStr, err = m.client.RPop(ctx, globalKey).Result()
	if err == nil {
		// Push to processing queue
		if err := m.client.LPush(ctx, processingKey, dataStr).Err(); err != nil {
			// Push back to global queue if failed
			m.client.RPush(ctx, globalKey, dataStr)
			return err
		}

		// Decode and call after process push functions
		var data T
		if err := m.unmarshalFunc([]byte(dataStr), &data); err == nil {
			for _, fn := range m.afterProcessPushFunc {
				fn(key, data)
			}
		}
	}

	return nil
}

func (m *QueueManager[T, K]) Close() error {
	return m.client.Close()
}
