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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/openimsdk/tools/db/redisutil"
	"github.com/openimsdk/tools/queue/task"
	"github.com/stretchr/testify/assert"
)

func getTestRedisConfig() *redisutil.Config {
	// Use environment variables for test Redis configuration
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	password := os.Getenv("REDIS_PASSWORD")
	db := 0 // Use database 0 for tests

	return &redisutil.Config{
		RedisMode: redisutil.StandaloneMode,
		Address:   []string{fmt.Sprintf("%s:%s", host, port)},
		Password:  password,
		DB:        db,
		MaxRetry:  3,
		PoolSize:  10,
	}
}

func TestRedisQueueManager_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()
	config := getTestRedisConfig()

	// Check Redis connection
	if err := redisutil.Check(ctx, config); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	manager, err := NewQueueManager[string, string](
		ctx, config,
		5, 2, 2, // maxGlobal, maxProcessing, maxWaiting
		func(a, b string) bool { return a == b },
		WithNamespace[string, string]("test_basic"),
	)
	assert.NoError(t, err)
	redisManager := manager.(*QueueManager[string, string])
	defer redisManager.Close()

	// Clear any existing data
	redisManager.client.FlushDB(ctx)

	// Test AddKey
	err = manager.AddKey(ctx, "user1")
	assert.NoError(t, err)
	err = manager.AddKey(ctx, "user2")
	assert.NoError(t, err)

	// Test Insert with auto-assignment
	err = manager.Insert(ctx, "task1")
	assert.NoError(t, err)

	err = manager.Insert(ctx, "task2")
	assert.NoError(t, err)

	// Verify processing queue lengths
	lengths, err := manager.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(lengths))
	assert.Equal(t, 1, lengths["user1"])
	assert.Equal(t, 1, lengths["user2"])

	// Test InsertByKey
	err = manager.InsertByKey(ctx, "user1", "task3")
	assert.NoError(t, err)

	// Processing queue should be full now for user1
	err = manager.InsertByKey(ctx, "user1", "task4")
	assert.NoError(t, err) // Should go to waiting queue

	err = manager.InsertByKey(ctx, "user1", "task5")
	assert.NoError(t, err) // Should go to waiting queue

	// Both queues full, should fail
	err = manager.InsertByKey(ctx, "user1", "task6")
	assert.Equal(t, task.ErrWaitingQueueFull, err)

	// Test Delete
	err = manager.Delete(ctx, "user1", "task1")
	assert.NoError(t, err)

	// Should backfill from waiting queue
	lengths, err = manager.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, lengths["user1"]) // Should still be 2 due to backfill

	// Test DeleteKey
	err = manager.DeleteKey(ctx, "user2")
	assert.NoError(t, err)
	lengths, err = manager.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(lengths))
	_, exists := lengths["user2"]
	assert.False(t, exists)
}

func TestRedisQueueManager_TransformProcessingData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()
	config := getTestRedisConfig()

	if err := redisutil.Check(ctx, config); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	manager, err := NewQueueManager[string, string](
		ctx, config,
		10, 3, 3, // maxGlobal, maxProcessing, maxWaiting
		func(a, b string) bool { return a == b },
		WithNamespace[string, string]("test_transform"),
	)
	assert.NoError(t, err)
	redisManager := manager.(*QueueManager[string, string])
	defer redisManager.Close()

	// Clear any existing data
	redisManager.client.FlushDB(ctx)

	err = manager.AddKey("user1")
	assert.NoError(t, err)
	err = manager.AddKey("user2")
	assert.NoError(t, err)

	// Add data to user1
	manager.InsertByKey(ctx, "user1", "task1")
	manager.InsertByKey(ctx, "user1", "task2")

	// Transform data from user1 to user2
	err = manager.TransformProcessingData(ctx, "user1", "user2", "task1")
	assert.NoError(t, err)

	lengths, err := manager.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, lengths["user1"])
	assert.Equal(t, 1, lengths["user2"])
}

func TestRedisQueueManager_GlobalQueue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()
	config := getTestRedisConfig()

	if err := redisutil.Check(ctx, config); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	manager, err := NewQueueManager[string, string](
		ctx, config,
		2, 1, 1, // maxGlobal, maxProcessing, maxWaiting
		func(a, b string) bool { return a == b },
		WithNamespace[string, string]("test_global"),
	)
	assert.NoError(t, err)
	redisManager := manager.(*QueueManager[string, string])
	defer redisManager.Close()

	// Clear any existing data
	redisManager.client.FlushDB(ctx)

	err = manager.AddKey(ctx, "user1")
	assert.NoError(t, err)

	// Fill processing queue
	err = manager.Insert(ctx, "task1")
	assert.NoError(t, err)

	// These should go to global queue
	err = manager.Insert(ctx, "task2")
	assert.NoError(t, err)

	err = manager.Insert(ctx, "task3")
	assert.NoError(t, err)

	// Global queue should be full
	err = manager.Insert(ctx, "task4")
	assert.Equal(t, task.ErrGlobalQueueFull, err)

	// Delete from processing queue should backfill from global
	err = manager.Delete(ctx, "user1", "task1")
	assert.NoError(t, err)

	lengths, err := manager.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, lengths["user1"]) // Should be backfilled
}

func TestRedisQueueManager_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()
	config := getTestRedisConfig()

	if err := redisutil.Check(ctx, config); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	manager, err := NewQueueManager[int, string](
		ctx, config,
		100, 10, 10, // maxGlobal, maxProcessing, maxWaiting
		func(a, b int) bool { return a == b },
		WithNamespace[int, string]("test_concurrent"),
	)
	assert.NoError(t, err)
	redisManager := manager.(*QueueManager[int, string])
	defer redisManager.Close()

	// Clear any existing data
	redisManager.client.FlushDB(ctx)

	// Add keys
	for i := 0; i < 5; i++ {
		err = manager.AddKey(ctx, fmt.Sprintf("user%d", i))
		assert.NoError(t, err)
	}

	// Concurrent inserts
	var wg sync.WaitGroup
	errors := make([]error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = manager.Insert(ctx, idx)
		}(i)
	}

	wg.Wait()

	// Check no errors
	for _, err := range errors {
		assert.NoError(t, err)
	}

	// Verify total items
	totalItems := 0
	lengths, err := manager.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	for _, length := range lengths {
		totalItems += length
	}
	assert.LessOrEqual(t, totalItems, 50) // Max 10 per queue * 5 queues
}

func TestRedisQueueManager_CustomFunctions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()
	config := getTestRedisConfig()

	if err := redisutil.Check(ctx, config); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	type CustomData struct {
		ID   string
		Name string
	}

	equalFunc := func(a, b CustomData) bool {
		return a.ID == b.ID
	}

	callCount := 0
	afterPushFunc := func(key string, data CustomData) {
		callCount++
	}

	manager, err := NewQueueManager[CustomData, string](
		ctx, config,
		10, 5, 5, // maxGlobal, maxProcessing, maxWaiting
		equalFunc,
		WithNamespace[CustomData, string]("test_custom"),
		WithAfterProcessPushFunc[CustomData, string](afterPushFunc),
	)
	assert.NoError(t, err)
	redisManager := manager.(*QueueManager[CustomData, string])
	defer redisManager.Close()

	// Clear any existing data
	redisManager.client.FlushDB(ctx)

	err = manager.AddKey(ctx, "user1")
	assert.NoError(t, err)

	// Insert and verify callback
	data := CustomData{ID: "1", Name: "Test"}
	err = manager.InsertByKey(ctx, "user1", data)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Delete using custom equal function
	err = manager.Delete(ctx, "user1", CustomData{ID: "1", Name: "Different"})
	assert.NoError(t, err)
}

func TestRedisQueueManager_Strategies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()
	config := getTestRedisConfig()

	if err := redisutil.Check(ctx, config); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Test Least Tasks strategy
	manager, err := NewQueueManager[string, string](
		ctx, config,
		100, 10, 10, // maxGlobal, maxProcessing, maxWaiting
		func(a, b string) bool { return a == b },
		WithNamespace[string, string]("test_strategy"),
		WithStrategy[string, string](Least),
	)
	assert.NoError(t, err)
	redisManager := manager.(*QueueManager[string, string])
	defer redisManager.Close()

	// Clear any existing data
	redisManager.client.FlushDB(ctx)

	err = manager.AddKey("user1")
	assert.NoError(t, err)
	err = manager.AddKey("user2")
	assert.NoError(t, err)
	err = manager.AddKey(ctx, "user3")
	assert.NoError(t, err)

	// Manually add tasks to create imbalance
	manager.InsertByKey(ctx, "user1", "task1")
	manager.InsertByKey(ctx, "user1", "task2")
	manager.InsertByKey(ctx, "user2", "task3")

	// New inserts should go to user3 (least tasks)
	for i := 0; i < 3; i++ {
		err = manager.Insert(ctx, fmt.Sprintf("auto_task_%d", i))
		assert.NoError(t, err)
	}

	lengths, err := manager.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, lengths["user3"]) // Should have received all auto-assigned tasks
}

func TestRedisQueueManager_MultiProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()
	config := getTestRedisConfig()

	if err := redisutil.Check(ctx, config); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	namespace := fmt.Sprintf("test_multiprocess_%d", time.Now().UnixNano())

	// Create two managers simulating different processes
	manager1, err := NewQueueManager[string, string](
		ctx, config,
		20, 5, 5, // maxGlobal, maxProcessing, maxWaiting
		func(a, b string) bool { return a == b },
		WithNamespace[string, string](namespace),
	)
	assert.NoError(t, err)
	redisManager1 := manager1.(*QueueManager[string, string])
	defer redisManager1.Close()

	manager2, err := NewQueueManager[string, string](
		ctx, config,
		20, 5, 5, // maxGlobal, maxProcessing, maxWaiting
		func(a, b string) bool { return a == b },
		WithNamespace[string, string](namespace),
	)
	assert.NoError(t, err)
	redisManager2 := manager2.(*QueueManager[string, string])
	defer redisManager2.Close()

	// Clear any existing data
	redisManager1.client.FlushDB(ctx)

	// Both managers add keys
	err = manager1.AddKey(ctx, "shared_key")
	assert.NoError(t, err)
	err = manager2.AddKey(ctx, "shared_key") // Should not duplicate
	assert.NoError(t, err)

	// Both insert data
	err = manager1.InsertByKey(ctx, "shared_key", "task_from_1")
	assert.NoError(t, err)

	err = manager2.InsertByKey(ctx, "shared_key", "task_from_2")
	assert.NoError(t, err)

	// Both should see the same queue lengths
	lengths1, err := manager1.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)
	lengths2, err := manager2.GetProcessingQueueLengths(ctx)
	assert.NoError(t, err)

	assert.Equal(t, lengths1, lengths2)
	assert.Equal(t, 2, lengths1["shared_key"])
}
