package disable

import (
	"context"
	"errors"
	"time"

	"github.com/openimsdk/tools/s3"
)

var errDisabled = errors.New("s3 disabled")

func NewDisable() s3.Interface {
	return disableS3{}
}

type disableS3 struct {
}

func (disableS3) Engine() string {
	return "disable"
}

func (disableS3) PartLimit() (*s3.PartLimit, error) {
	return nil, errDisabled
}

func (disableS3) InitiateMultipartUpload(ctx context.Context, name string) (*s3.InitiateMultipartUploadResult, error) {
	return nil, errDisabled
}

func (disableS3) CompleteMultipartUpload(ctx context.Context, uploadID string, name string, parts []s3.Part) (*s3.CompleteMultipartUploadResult, error) {
	return nil, errDisabled
}

func (disableS3) PartSize(ctx context.Context, size int64) (int64, error) {
	return 0, errDisabled
}

func (disableS3) AuthSign(ctx context.Context, uploadID string, name string, expire time.Duration, partNumbers []int) (*s3.AuthSignResult, error) {
	return nil, errDisabled
}

func (disableS3) PresignedPutObject(ctx context.Context, name string, expire time.Duration) (string, error) {
	return "", errDisabled
}

func (disableS3) DeleteObject(ctx context.Context, name string) error {
	return errDisabled
}

func (disableS3) CopyObject(ctx context.Context, src string, dst string) (*s3.CopyObjectInfo, error) {
	return nil, errDisabled
}

func (disableS3) StatObject(ctx context.Context, name string) (*s3.ObjectInfo, error) {
	return nil, errDisabled
}

func (disableS3) IsNotFound(err error) bool {
	return false
}

func (disableS3) AbortMultipartUpload(ctx context.Context, uploadID string, name string) error {
	return errDisabled
}

func (disableS3) ListUploadedParts(ctx context.Context, uploadID string, name string, partNumberMarker int, maxParts int) (*s3.ListUploadedPartsResult, error) {
	return nil, errDisabled
}

func (disableS3) AccessURL(ctx context.Context, name string, expire time.Duration, opt *s3.AccessURLOption) (string, error) {
	return "", errDisabled
}

func (disableS3) FormData(ctx context.Context, name string, size int64, contentType string, duration time.Duration) (*s3.FormData, error) {
	return nil, errDisabled
}
