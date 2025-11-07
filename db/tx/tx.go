package tx

import "context"

type Tx interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
