package task

// QueueManager defines the interface for managing task queues
type QueueManager[T any, K comparable] interface {
	// AddKey adds a new key to the queue manager
	AddKey(key K)

	// Insert inserts data into the queue, automatically assigning a key based on the strategy
	Insert(data T) error

	// InsertByKey inserts data into the queue for a specific key
	InsertByKey(key K, data T) error

	// Delete removes data from the specified key's queue
	Delete(key K, data T) error

	// DeleteKey removes a key and its associated queue
	DeleteKey(key K)

	// GetProcessingQueueLengths returns the length of processing queue for each key
	GetProcessingQueueLengths() map[K]int

	// TransformProcessingData moves data from one key's processing queue to another
	TransformProcessingData(fromKey, toKey K, data T)
}
