package task

import "context"

// QueueManager defines the interface for managing task queues
type QueueManager[T any, K comparable] interface {
	// AddKey adds a new key to the queue manager
	AddKey(ctx context.Context, key K) error

	// Insert inserts data into the queue, automatically assigning a key based on the strategy
	// Returns the assigned key (empty string if queued in global queue)
	Insert(ctx context.Context, data T) (K, error)

	// InsertByKey inserts data into the queue for a specific key
	InsertByKey(ctx context.Context, key K, data T) error

	// Delete removes data from the specified key's queue
	Delete(ctx context.Context, key K, data T) error

	// DeleteKey removes a key and its associated queue
	DeleteKey(ctx context.Context, key K) error

	// GetProcessingQueueLengths returns the length of processing queue for each key
	GetProcessingQueueLengths(ctx context.Context) (map[K]int, error)

	// TransformProcessingData moves data from one key's processing queue to another
	TransformProcessingData(ctx context.Context, fromKey, toKey K, data T) error

	// AutoTransformProcessingData moves data from one key's processing queue to another key selected by strategy
	// Returns the selected target key
	AutoTransformProcessingData(ctx context.Context, fromKey K, data T) (K, error)

	// GetGlobalQueuePosition returns the position of data in the global queue (0-based, -1 if not found)
	GetGlobalQueuePosition(ctx context.Context, data T) (int, error)
}
