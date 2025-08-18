package timer

import (
	"context"
	"time"
)

// KeyFunc extracts a unique key from an item
type KeyFunc[T any] func(item T) string

// Handler processes items when their timer expires
type Handler[T any] func(ctx context.Context, item T) error

// HandlerMap maps timer types to their handlers
type HandlerMap[T any] map[string]Handler[T]

// Manager manages timers for items
type Manager[T any] interface {
	// Register adds an item with a timeout duration for a specific type
	// If a timer with the same key already exists, it will be updated (upsert behavior)
	Register(ctx context.Context, timerType string, item T, timeout time.Duration) error

	// RegisterAt adds an item that expires at a specific time for a specific type
	RegisterAt(ctx context.Context, timerType string, item T, expireAt time.Time) error

	// Cancel removes a timer for an item of a specific type
	Cancel(ctx context.Context, timerType string, key string) error

	// GetPending returns the count of pending timers for a specific type
	GetPending(ctx context.Context, timerType string) (int64, error)

	// Close releases resources and stops processing
	Close() error
}
