package log

const (
	slicePrintLen = 30
)

type Slice[T any] []T

func (s Slice[T]) Format() any {
	if len(s) >= slicePrintLen {
		return s[0:slicePrintLen]
	}
	return s
}
