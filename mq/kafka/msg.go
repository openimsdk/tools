package kafka

import (
	"context"
)

type kafkaMessage struct {
	ctx context.Context
	msg *consumerMessage
}

func (m kafkaMessage) Context() context.Context {
	return m.ctx
}

func (m kafkaMessage) Key() string {
	return string(m.msg.Msg.Key)
}

func (m kafkaMessage) Value() []byte {
	return m.msg.Msg.Value
}

func (m kafkaMessage) Mark() {
	m.msg.Session.MarkMessage(m.msg.Msg, "")
}

func (m kafkaMessage) Commit() {
	m.msg.Session.Commit()
}
