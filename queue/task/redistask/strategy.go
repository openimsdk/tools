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
	"fmt"

	"github.com/redis/go-redis/v9"
)

// strategy: assign key strategy
type strategy int

const (
	RoundRobin strategy = iota + 1
	Least
)

func getStrategy[T any, K comparable](s strategy) func(ctx context.Context, m *QueueManager[T, K]) (K, bool) {
	switch s {
	case RoundRobin:
		return RoundRobinTask[T, K]
	case Least:
		return LeastTask[T, K]
	}
	return nil
}

// RoundRobinTask : return key in round-robin fashion
func RoundRobinTask[T any, K comparable](ctx context.Context, m *QueueManager[T, K]) (K, bool) {
	// Lua script to atomically find next available key
	script := redis.NewScript(`
		local keys_list = KEYS[1]
		local last_index_key = KEYS[2]
		local max_processing = tonumber(ARGV[1])
		local namespace = ARGV[2]
		
		-- Get all keys
		local keys = redis.call('LRANGE', keys_list, 0, -1)
		if #keys == 0 then
			return nil
		end
		
		-- Get last assigned index
		local last_index = redis.call('GET', last_index_key)
		if not last_index then
			last_index = -1
		else
			last_index = tonumber(last_index)
		end
		
		-- Try each key starting from next position
		for i = 1, #keys do
			local index = (last_index + i) % #keys
			local key = keys[index + 1]  -- Lua arrays are 1-indexed
			
			-- Check if processing queue has space
			local processing_key = string.format("taskqueue:%s:processing:%s", namespace, key)
			local queue_length = redis.call('LLEN', processing_key)
			
			if queue_length < max_processing then
				-- Update last assigned index
				redis.call('SET', last_index_key, index)
				return key
			end
		end
		
		-- All queues are full
		return nil
	`)

	result, err := script.Run(ctx, m.client,
		[]string{m.getKeysListKey(), m.getLastIndexKey()},
		m.maxProcessing, m.namespace).Result()

	if err != nil || result == nil {
		var zero K
		return zero, false
	}

	// Parse the result
	keyStr, ok := result.(string)
	if !ok {
		var zero K
		return zero, false
	}

	var key K
	if _, err := fmt.Sscanf(keyStr, "%v", &key); err != nil {
		var zero K
		return zero, false
	}

	return key, true
}

// LeastTask : return key which has the least tasks
func LeastTask[T any, K comparable](ctx context.Context, m *QueueManager[T, K]) (K, bool) {
	// Get all keys from Redis list
	keyStrs, err := m.client.LRange(ctx, m.getKeysListKey(), 0, -1).Result()
	if err != nil || len(keyStrs) == 0 {
		var zero K
		return zero, false
	}

	var minKey K
	minTasks := int64(m.maxProcessing + 1)
	found := false

	// Check each key's processing queue length
	for _, keyStr := range keyStrs {
		var key K
		if _, err := fmt.Sscanf(keyStr, "%v", &key); err != nil {
			continue
		}

		length, err := m.client.LLen(ctx, m.getProcessingQueueKey(key)).Result()
		if err == nil && length < int64(m.maxProcessing) {
			if length < minTasks {
				minKey = key
				minTasks = length
				found = true
			}
		}
	}

	return minKey, found
}
