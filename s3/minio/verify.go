package minio

import "context"

func Check(ctx context.Context, config *Config) error {
	m, err := NewMinio(ctx, nil, *config)
	if err != nil {
		return err
	}
	return m.initMinio(ctx)
}
