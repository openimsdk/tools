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

type Map[K comparable, V any] map[K]V

func (m Map[K, V]) Format() any {
	if len(m) > slicePrintLen {
		out := make(map[K]V, slicePrintLen)
		i := 0
		for k, v := range m {
			out[k] = v
			i++
			if i >= slicePrintLen {
				break
			}
		}
		return out
	}
	return m
}
