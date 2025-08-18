package standalonetask

type Options[T any, K comparable] func(*QueueManager[T, K])

func WithStrategy[T any, K comparable](s strategy) Options[T, K] {
	return func(tm *QueueManager[T, K]) {
		tm.assignStrategy = getStrategy[T, K](s)
	}
}

func WithAfterProcessPushFunc[T any, K comparable](fs ...func(key K, data T)) Options[T, K] {
	return func(tm *QueueManager[T, K]) {
		tm.afterProcessPushFunc = fs
	}
}
