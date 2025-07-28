package mq

import "context"

type Message interface {
	Context() context.Context
	Key() string
	Value() []byte
	Mark()
	Commit()
}

type Handler func(msg Message) error

type Consumer interface {
	Subscribe(ctx context.Context, fn Handler) error
	Close() error
}

type Producer interface {
	SendMessage(ctx context.Context, key string, value []byte) error
	Close() error
}
