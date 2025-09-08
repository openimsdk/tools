package bound

import (
	"errors"
	"sync"
)

var (
	ErrQueueFull  = errors.New("queue is full")
	ErrQueueEmpty = errors.New("queue is empty")
)

// Queue has its capacity. Push returns an error when the Queue is full.
type Queue[T any] struct {
	items    []T
	capacity int
	lock     sync.RWMutex
}

func NewQueue[T any](capacity int) *Queue[T] {
	return &Queue[T]{
		items:    make([]T, 0, capacity),
		capacity: capacity,
	}
}

func (q *Queue[T]) Full() bool {
	return q.Len() >= q.capacity
}

func (q *Queue[T]) Push(item T) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.items) >= q.capacity {
		return ErrQueueFull
	}
	q.items = append(q.items, item)
	return nil
}

// ForcePush pushes an item to the queue regardless of capacity.
// If the queue is at capacity, it will exceed the capacity limit.
func (q *Queue[T]) ForcePush(item T) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.items = append(q.items, item)
}

func (q *Queue[T]) Pop() (T, error) {
	q.lock.Lock()
	defer q.lock.Unlock()
	var zero T
	if len(q.items) == 0 {
		return zero, ErrQueueEmpty
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, nil
}

func (q *Queue[T]) Len() int {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return len(q.items)
}

func (q *Queue[T]) Contains(item T, equalFunc func(a, b T) bool) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()
	for _, it := range q.items {
		if equalFunc(it, item) {
			return true
		}
	}
	return false
}

func (q *Queue[T]) Remove(item T, equalFunc func(a, b T) bool) bool {
	q.lock.Lock()
	defer q.lock.Unlock()
	for i, it := range q.items {
		if equalFunc(it, item) {
			q.items = append(q.items[:i], q.items[i+1:]...)
			return true
		}
	}
	return false
}

func (q *Queue[T]) Peek(item T, equalFunc func(a, b T) bool) int {
	q.lock.RLock()
	defer q.lock.RUnlock()
	for i, it := range q.items {
		if equalFunc(it, item) {
			return i
		}
	}
	return -1
}
