package mq

import "context"

type Handler func(ctx context.Context, key string, value []byte) error

type Consumer interface {
	Subscribe(ctx context.Context, fn Handler) error
	Close() error
}

type Producer interface {
	SendMessage(ctx context.Context, key string, value []byte) error
	Close() error
}
