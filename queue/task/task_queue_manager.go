package task

import (
	"errors"
	"sync"

	"github.com/openimsdk/tools/queue/bound"
)

var (
	ErrGlobalQueueFull     = errors.New("global Queue is full")
	ErrWaitingQueueFull    = errors.New("waiting Queue is full")
	ErrProcessingQueueFull = errors.New("processing Queue is full")
	ErrDataNotFound        = errors.New("data not found")
)

// Queue will pop data from its waiting Queue. If it`s empty, it will pop data from global Queue(in QueueManager),
// and then push to process Queue.
type Queue[T any] struct {
	processing *bound.BoundedQueue[T]
	waiting    *bound.BoundedQueue[T]
}

type QueueManager[T any, K comparable] struct {
	globalQueue   *bound.BoundedQueue[T]
	taskQueues    map[K]*Queue[T]
	maxProcessing int
	maxWaiting    int
	lock          sync.RWMutex
	equalDataFunc func(a, b T) bool

	afterProcessPushFunc []func(key K, data T) // will be called after processing Queue push successfully

	// assign key strategy. Determine witch key will be assigned when push data without assigning a key
	assignStrategy func(*QueueManager[T, K]) (K, bool)
}

func NewQueueManager[T any, K comparable](
	maxGlobal, maxProcessing, maxWaiting int,
	equalFunc func(a, b T) bool,
	opts ...Options[T, K],
) *QueueManager[T, K] {
	tm := &QueueManager[T, K]{
		globalQueue:    bound.NewBoundedQueue[T](maxGlobal),
		taskQueues:     make(map[K]*Queue[T]),
		maxProcessing:  maxProcessing,
		maxWaiting:     maxWaiting,
		equalDataFunc:  equalFunc,
		assignStrategy: getStrategy[T, K](Least),
	}

	tm.applyOpts(opts)

	return tm
}

func (tm *QueueManager[T, K]) applyOpts(opts []Options[T, K]) {
	for _, opt := range opts {
		opt(tm)
	}
}

func (tm *QueueManager[T, K]) getOrCreateTaskQueues(k K) *Queue[T] {
	if q, exists := tm.taskQueues[k]; exists {
		return q
	}
	q := &Queue[T]{
		processing: bound.NewBoundedQueue[T](tm.maxProcessing),
		waiting:    bound.NewBoundedQueue[T](tm.maxWaiting),
	}
	tm.taskQueues[k] = q
	return q
}

func (tm *QueueManager[T, K]) assignKey() (K, bool) {
	return tm.assignStrategy(tm)

}

func (tm *QueueManager[T, K]) Insert(data T) error {
	tm.lock.Lock()
	k, assigned := tm.assignKey()
	defer tm.lock.Unlock()

	if !assigned {
		if !tm.globalQueue.Full() {
			return tm.globalQueue.Push(data)
		}
		return ErrGlobalQueueFull
	}

	taskQueues := tm.taskQueues[k]
	if !taskQueues.processing.Full() {
		return taskQueues.processing.Push(data)
	}

	if !tm.globalQueue.Full() {
		return tm.globalQueue.Push(data)
	}

	return ErrGlobalQueueFull
}

func (tm *QueueManager[T, K]) InsertByKey(key K, data T) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	taskQueues := tm.getOrCreateTaskQueues(key)

	if !taskQueues.processing.Full() {
		return tm.pushToProcess(taskQueues, key, data)
	}

	if !taskQueues.waiting.Full() {
		return taskQueues.waiting.Push(data)
	}

	return ErrWaitingQueueFull
}

// Delete will delete a data in key queues. If delete a data in processing Queue, taskQueue will pop data from its
// waiting Queue. If it`s empty, it will pop data from global Queue, and then push to process Queue.
func (tm *QueueManager[T, K]) Delete(key K, data T) error {
	tm.lock.Lock()
	taskQueue, exists := tm.taskQueues[key]
	tm.lock.Unlock()
	if !exists {
		return ErrDataNotFound
	}

	if removed := taskQueue.processing.Remove(data, tm.equalDataFunc); removed {
		if nextData, err := taskQueue.waiting.Pop(); err == nil {
			_ = tm.pushToProcess(taskQueue, key, nextData)
		} else {
			if globalData, err := tm.globalQueue.Pop(); err == nil {
				_ = tm.pushToProcess(taskQueue, key, globalData)
			}
		}
		return nil
	}

	// try removing data in waiting Queue
	if removed := taskQueue.waiting.Remove(data, tm.equalDataFunc); removed {
		return nil
	}

	return ErrDataNotFound
}

func (tm *QueueManager[T, K]) GetProcessingQueueLengths() map[K]int {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	lengths := make(map[K]int)
	for id, q := range tm.taskQueues {
		lengths[id] = q.processing.Len()
	}
	return lengths
}

func (tm *QueueManager[T, K]) pushToProcess(taskQueues *Queue[T], key K, data T) error {
	err := taskQueues.processing.Push(data)
	if err != nil {
		return err
	}
	for _, f := range tm.afterProcessPushFunc {
		f(key, data)
	}
	return nil
}
