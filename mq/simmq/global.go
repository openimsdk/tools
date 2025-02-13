package simmq

import (
	"sync"

	"github.com/openimsdk/tools/mq"
)

const defaultMqSize = 1024 * 16

var (
	topicMq   map[string]*memory
	topicLock sync.Mutex
)

func getTopicMemory(topic string) *memory {
	topicLock.Lock()
	defer topicLock.Unlock()
	if topicMq == nil {
		topicMq = make(map[string]*memory)
	}
	val, ok := topicMq[topic]
	if !ok {
		val = newMemory(defaultMqSize, func() {
			topicLock.Lock()
			delete(topicMq, topic)
			topicLock.Unlock()
		})
		topicMq[topic] = val
	}
	return val
}

func GetTopicProducer(topic string) mq.Producer {
	return getTopicMemory(topic)
}

func GetTopicConsumer(topic string) mq.Consumer {
	return getTopicMemory(topic)
}
