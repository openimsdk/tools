package standalonetask

// strategy: assign key strategy
type strategy int

const (
	RoundRobin strategy = iota + 1
	Least
)

func getStrategy[T any, K comparable](s strategy) func(tm *QueueManager[T, K]) (K, bool) {
	switch s {
	case RoundRobin:
		return RoundRobinTask[T, K]
	case Least:
		return LeastTask[T, K]
	}
	return nil
}

// RoundRobinTask : return key in round-robin fashion
func RoundRobinTask[T any, K comparable](tm *QueueManager[T, K]) (K, bool) {
	if len(tm.orderedKeys) == 0 {
		var zero K
		return zero, false
	}

	// Find next available queue that's not full
	startIndex := tm.lastAssignedIndex
	for i := 0; i < len(tm.orderedKeys); i++ {
		index := (startIndex + i + 1) % len(tm.orderedKeys)
		key := tm.orderedKeys[index]

		if queue, exists := tm.taskQueues[key]; exists {
			if !queue.processing.Full() {
				tm.lastAssignedIndex = index
				return key, true
			}
		}
	}

	// All queues are full
	var zero K
	return zero, false
}

// LeastTask : return key witch has the least tasks
func LeastTask[T any, K comparable](tm *QueueManager[T, K]) (K, bool) {
	var k K
	minLen := -1
	for id, q := range tm.taskQueues {
		length := q.processing.Len()
		if minLen == -1 || length < minLen {
			minLen = length
			k = id
		}
	}
	return k, minLen != -1
}
