package minio

import "context"

type ImageInfo struct {
	IsImg  bool   `json:"isImg"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
	Etag   string `json:"etag"`
}

type Cache interface {
	GetImageObjectKeyInfo(ctx context.Context, key string, fn func(ctx context.Context) (*ImageInfo, error)) (*ImageInfo, error)
	GetThumbnailKey(ctx context.Context, key string, format string, width int, height int, minioCache func(ctx context.Context) (string, error)) (string, error)
	DelObjectImageInfoKey(ctx context.Context, keys ...string) error
	DelImageThumbnailKey(ctx context.Context, key string, format string, width int, height int) error
}
