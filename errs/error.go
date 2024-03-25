package errs

import "github.com/pkg/errors"

type Error interface {
	Is(err error) bool
	Wrap() error
	WrapMsg(msg string, kv ...any) error
	error
}

func New(s string) Error {
	return &errorString{
		s: s,
	}
}

type errorString struct {
	s string
}

func (e *errorString) Is(err error) bool {
	return errors.Is(e, err)
}

func (e *errorString) Error() string {
	return e.s
}

func (e *errorString) Wrap() error {
	return Wrap(e)
}

func (e *errorString) WrapMsg(msg string, kv ...any) error {
	return WrapMsg(e, msg, kv...)
}
