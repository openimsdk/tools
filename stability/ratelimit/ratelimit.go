package ratelimit

import "errors"

var (
	ErrLimitExceeded = errors.New("ratelimit: limit exceeded")
)

type DoneFunc func(DoneInfo)

type DoneInfo struct {
	Err error
}

type Limiter interface {
	Allow() (DoneFunc, error)
}
