package simmq

import "context"

type message struct {
	ctx   context.Context
	key   string
	value []byte
}

func (m *message) Context() context.Context {
	return m.ctx
}

func (m *message) Key() string {
	return m.key
}

func (m *message) Value() []byte {
	return m.value
}

func (m *message) Mark() {
}

func (m *message) Commit() {
}
