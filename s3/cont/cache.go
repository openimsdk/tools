package cont

import (
	"context"
	"github.com/openimsdk/tools/s3"
)

type S3Cache interface {
	GetKey(ctx context.Context, engine string, key string) (*s3.ObjectInfo, error)
	DelS3Key(ctx context.Context, engine string, keys ...string) error
}
