package bound

import (
	"errors"
	"sync"
)

var (
	ErrQueueFull  = errors.New("queue is full")
	ErrQueueEmpty = errors.New("queue is empty")
)

// BoundedQueue has its capacity. Push returns an error when the BoundedQueue is full.
type BoundedQueue[T any] struct {
	items    []T
	capacity int
	lock     sync.RWMutex
}

func NewBoundedQueue[T any](capacity int) *BoundedQueue[T] {
	return &BoundedQueue[T]{
		items:    make([]T, 0, capacity),
		capacity: capacity,
	}
}

func (q *BoundedQueue[T]) Full() bool {
	return q.Len() >= q.capacity
}

func (q *BoundedQueue[T]) Push(item T) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.items) >= q.capacity {
		return ErrQueueFull
	}
	q.items = append(q.items, item)
	return nil
}

func (q *BoundedQueue[T]) Pop() (T, error) {
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

func (q *BoundedQueue[T]) Len() int {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return len(q.items)
}

func (q *BoundedQueue[T]) Contains(item T, equalFunc func(a, b T) bool) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()
	for _, it := range q.items {
		if equalFunc(it, item) {
			return true
		}
	}
	return false
}

func (q *BoundedQueue[T]) Remove(item T, equalFunc func(a, b T) bool) bool {
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
