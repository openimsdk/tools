package simmq

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/openimsdk/tools/mq"
)

var (
	errClosed = errors.New("memory mq closed")
)

type message struct {
	ctx   context.Context
	key   string
	value []byte
}

func NewMemory(size int) (mq.Producer, mq.Consumer) {
	m := newMemory(size, nil)
	return m, m
}

func newMemory(size int, fn func()) *memory {
	return &memory{
		ch:   make(chan *message, size),
		done: make(chan struct{}),
		fn:   fn,
	}
}

type memory struct {
	lock   sync.RWMutex
	closed atomic.Bool
	ch     chan *message
	done   chan struct{}
	fn     func()
}

func (x *memory) Subscribe(ctx context.Context, fn mq.Handler) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case msg, ok := <-x.ch:
		if !ok {
			return errClosed
		}
		if err := fn(msg.ctx, msg.key, msg.value); err != nil {
			return err
		}
		return nil
	}
}

func (x *memory) SendMessage(ctx context.Context, key string, value []byte) error {
	if x.closed.Load() {
		return errClosed
	}
	msg := &message{
		ctx:   context.WithoutCancel(ctx),
		key:   key,
		value: value,
	}
	x.lock.RLock()
	defer x.lock.RUnlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-x.done:
		return errClosed
	case x.ch <- msg:
		return nil
	}
}

func (x *memory) Close() error {
	if !x.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(x.done)
	if x.fn != nil {
		x.fn()
	}
	x.lock.Lock()
	defer x.lock.Unlock()
	close(x.ch)
	return nil
}
