package task

// strategy: assign key strategy
type strategy int

const (
	Least strategy = iota + 1
)

func getStrategy[T any, K comparable](s strategy) func(tm *QueueManager[T, K]) (K, bool) {
	switch s {
	case Least:
		return LeastTask[T, K]
	}
	return nil
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
