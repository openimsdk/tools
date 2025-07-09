package gcp

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/openimsdk/tools/s3"
	"google.golang.org/api/option"
)

const (
	minPartSize int64 = 1024 * 1024 * 5
	maxPartSize int64 = 1024 * 1024 * 1024 * 5
	maxNumSize  int64 = 10000
)

type Config struct {
	Bucket         string
	CredentialJSON []byte
	GoogleAccessID string
	PrivateKey     []byte
}

type GCS struct {
	bucket         string
	client         *storage.Client
	googleAccessID string
	privateKey     []byte
}

func NewGCS(conf Config) (*GCS, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON(conf.CredentialJSON))
	if err != nil {
		return nil, err
	}

	if conf.GoogleAccessID == "" || len(conf.PrivateKey) == 0 {
		accessID, priKey, err := extractAccessIDAndKey(conf.CredentialJSON)
		if err != nil {
			return nil, err
		}
		conf.GoogleAccessID = accessID
		conf.PrivateKey = priKey
	}

	return &GCS{
		bucket:         conf.Bucket,
		client:         client,
		googleAccessID: conf.GoogleAccessID,
		privateKey:     conf.PrivateKey,
	}, nil
}

func extractAccessIDAndKey(jsonData []byte) (string, []byte, error) {
	var creds struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal(jsonData, &creds); err != nil {
		return "", nil, err
	}
	if creds.ClientEmail == "" || creds.PrivateKey == "" {
		return "", nil, errors.New("missing client_email or private_key in credentials")
	}
	return creds.ClientEmail, []byte(creds.PrivateKey), nil
}

func (g *GCS) Engine() string { return "google-cloud-storage" }

func (g *GCS) PartLimit() (*s3.PartLimit, error) {
	return &s3.PartLimit{
		MinPartSize: minPartSize,
		MaxPartSize: maxPartSize,
		MaxNumSize:  maxNumSize,
	}, nil
}

func (g *GCS) PartSize(ctx context.Context, size int64) (int64, error) {
	if size <= 0 {
		return 0, errors.New("size must be greater than 0")
	}
	if size > maxPartSize*maxNumSize {
		return 0, errors.New("gcs size exceeds max allowed")
	}
	if size <= minPartSize*maxNumSize {
		return minPartSize, nil
	}
	partSize := size / maxNumSize
	if size%maxNumSize != 0 {
		partSize++
	}
	return partSize, nil
}

func (g *GCS) IsNotFound(err error) bool {
	return errors.Is(err, storage.ErrObjectNotExist)
}

func (g *GCS) DeleteObject(ctx context.Context, name string) error {
	return g.client.Bucket(g.bucket).Object(name).Delete(ctx)
}

func (g *GCS) StatObject(ctx context.Context, name string) (*s3.ObjectInfo, error) {
	attrs, err := g.client.Bucket(g.bucket).Object(name).Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return &s3.ObjectInfo{
		ETag:         attrs.Etag,
		Key:          attrs.Name,
		Size:         attrs.Size,
		LastModified: attrs.Updated,
	}, nil
}

func (g *GCS) PresignedPutObject(ctx context.Context, name string, expire time.Duration, opt *s3.PutOption) (*s3.PresignedPutResult, error) {
	u, err := storage.SignedURL(g.bucket, name, &storage.SignedURLOptions{
		Method:         "PUT",
		Expires:        time.Now().Add(expire),
		GoogleAccessID: g.googleAccessID,
		PrivateKey:     g.privateKey,
		ContentType:    opt.ContentType,
	})
	if err != nil {
		return nil, err
	}
	return &s3.PresignedPutResult{URL: u}, nil
}

func (g *GCS) CopyObject(ctx context.Context, src, dst string) (*s3.CopyObjectInfo, error) {
	srcObj := g.client.Bucket(g.bucket).Object(src)
	dstObj := g.client.Bucket(g.bucket).Object(dst)
	_, err := dstObj.CopierFrom(srcObj).Run(ctx)
	if err != nil {
		return nil, err
	}
	attrs, err := dstObj.Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return &s3.CopyObjectInfo{Key: dst, ETag: attrs.Etag}, nil
}

func (g *GCS) AccessURL(ctx context.Context, name string, expire time.Duration, opt *s3.AccessURLOption) (string, error) {
	queryParams := make(url.Values)

	if opt.ContentType != "" {
		queryParams.Set("response-content-type", opt.ContentType)
	}
	if opt.Filename != "" {
		disposition := fmt.Sprintf(`attachment; filename="%s"`, opt.Filename)
		queryParams.Set("response-content-disposition", disposition)
	}
	return storage.SignedURL(g.bucket, name, &storage.SignedURLOptions{
		Method:          "GET",
		Expires:         time.Now().Add(expire),
		GoogleAccessID:  g.googleAccessID,
		PrivateKey:      g.privateKey,
		QueryParameters: queryParams,
	})
}

func (g *GCS) InitiateMultipartUpload(ctx context.Context, name string, opt *s3.PutOption) (*s3.InitiateMultipartUploadResult, error) {
	uploadID := fmt.Sprintf("upload-%d", time.Now().UnixNano())
	return &s3.InitiateMultipartUploadResult{
		Bucket:   g.bucket,
		Key:      name,
		UploadID: uploadID,
	}, nil
}

func (g *GCS) CompleteMultipartUpload(ctx context.Context, uploadID string, name string, parts []s3.Part) (*s3.CompleteMultipartUploadResult, error) {
	if len(parts) == 0 {
		return nil, errors.New("no parts to compose")
	}

	sources := make([]*storage.ObjectHandle, 0, len(parts))
	for _, part := range parts {
		objectName := fmt.Sprintf("%s_part_%d_%s", name, part.PartNumber, uploadID)
		sources = append(sources, g.client.Bucket(g.bucket).Object(objectName))
	}
	composer := g.client.Bucket(g.bucket).Object(name).ComposerFrom(sources...)
	attrs, err := composer.Run(ctx)
	if err != nil {
		return nil, err
	}

	return &s3.CompleteMultipartUploadResult{
		Location: fmt.Sprintf("gs://%s/%s", g.bucket, name),
		Bucket:   g.bucket,
		Key:      name,
		ETag:     attrs.Etag,
	}, nil
}

func (g *GCS) AbortMultipartUpload(ctx context.Context, uploadID string, name string) error {
	return nil // 不需要中止
}

func (g *GCS) ListUploadedParts(ctx context.Context, uploadID string, name string, partNumberMarker int, maxParts int) (*s3.ListUploadedPartsResult, error) {
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		Prefix: fmt.Sprintf("%s_part_", name),
	})

	res := &s3.ListUploadedPartsResult{
		Key:           name,
		UploadID:      uploadID,
		UploadedParts: []s3.UploadedPart{},
	}

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		if !strings.Contains(attrs.Name, uploadID) {
			continue
		}
		segments := strings.Split(attrs.Name, "_")
		if len(segments) < 4 {
			continue
		}
		partStr := segments[len(segments)-2]
		partNum, err := strconv.Atoi(partStr)
		if err != nil {
			continue
		}
		res.UploadedParts = append(res.UploadedParts, s3.UploadedPart{
			PartNumber:   partNum,
			LastModified: attrs.Updated,
			ETag:         attrs.Etag,
			Size:         attrs.Size,
		})
	}
	return res, nil
}

func (g *GCS) AuthSign(ctx context.Context, uploadID string, name string, expire time.Duration, partNumbers []int) (*s3.AuthSignResult, error) {
	res := &s3.AuthSignResult{
		Parts:  make([]s3.SignPart, 0, len(partNumbers)),
		Header: http.Header{},
		Query:  url.Values{},
	}

	for _, partNumber := range partNumbers {
		objectName := fmt.Sprintf("%s_part_%d_%s", name, partNumber, uploadID)
		signedURL, err := storage.SignedURL(g.bucket, objectName, &storage.SignedURLOptions{
			Method:         "PUT",
			Expires:        time.Now().Add(expire),
			GoogleAccessID: g.googleAccessID,
			PrivateKey:     g.privateKey,
		})
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(signedURL)
		if err != nil {
			return nil, err
		}
		query := u.Query()
		u.RawQuery = ""
		res.Parts = append(res.Parts, s3.SignPart{
			PartNumber: partNumber,
			URL:        u.String(),
			Query:      query,
			Header:     http.Header{},
		})
	}

	if len(res.Parts) > 0 {
		res.URL = res.Parts[0].URL
	}

	return res, nil
}

func (g *GCS) FormData(ctx context.Context, name string, size int64, contentType string, duration time.Duration) (*s3.FormData, error) {
	expires := time.Now().Add(duration)
	conditions := []storage.PostPolicyV4Condition{
		storage.ConditionContentLengthRange(0, uint64(size)),
	}

	opts := &storage.PostPolicyV4Options{
		GoogleAccessID: g.googleAccessID,
		PrivateKey:     g.privateKey,
		Expires:        expires,
		Fields: &storage.PolicyV4Fields{
			ContentType: contentType,
		},
		Conditions: conditions,
	}

	policy, err := storage.GenerateSignedPostPolicyV4(g.bucket, name, opts)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "multipart/form-data")

	return &s3.FormData{
		URL:          policy.URL,
		File:         "file",
		Header:       headers,
		FormData:     policy.Fields,
		Expires:      expires,
		SuccessCodes: []int{http.StatusOK, http.StatusCreated, http.StatusNoContent},
	}, nil
}
