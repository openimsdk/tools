// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kodo

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/qiniu/go-sdk/v7/storage"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/s3"
	"github.com/qiniu/go-sdk/v7/auth"
)

const (
	minPartSize = 1024 * 1024 * 1        // 1MB
	maxPartSize = 1024 * 1024 * 1024 * 5 // 5GB
	maxNumSize  = 10000
)

const successCode = http.StatusOK

type Config struct {
	Endpoint        string
	Bucket          string
	BucketURL       string
	AccessKeyID     string
	AccessKeySecret string
	SessionToken    string
	PublicRead      bool
}

type Kodo struct {
	AccessKey     string
	SecretKey     string
	Region        string
	Token         string
	Endpoint      string
	BucketURL     string
	Auth          *auth.Credentials
	Client        *awss3.Client
	PresignClient *awss3.PresignClient
}

func NewKodo(conf Config) (*Kodo, error) {
	//init client
	cfg, err := awss3config.LoadDefaultConfig(context.TODO(),
		awss3config.WithRegion(conf.Bucket),
		awss3config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: conf.Endpoint}, nil
			})),
		awss3config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			conf.AccessKeyID,
			conf.AccessKeySecret,
			conf.SessionToken),
		),
	)
	if err != nil {
		return nil, err
	}
	client := awss3.NewFromConfig(cfg)
	presignClient := awss3.NewPresignClient(client)

	return &Kodo{
		AccessKey:     conf.AccessKeyID,
		SecretKey:     conf.AccessKeySecret,
		Region:        conf.Bucket,
		BucketURL:     conf.BucketURL,
		Auth:          auth.New(conf.AccessKeyID, conf.AccessKeySecret),
		Client:        client,
		PresignClient: presignClient,
	}, nil
}

func (k *Kodo) Engine() string {
	return "kodo"
}

func (k *Kodo) PartLimit() *s3.PartLimit {
	return &s3.PartLimit{
		MinPartSize: minPartSize,
		MaxPartSize: maxPartSize,
		MaxNumSize:  maxNumSize,
	}
}

func (k *Kodo) InitiateMultipartUpload(ctx context.Context, name string) (*s3.InitiateMultipartUploadResult, error) {
	result, err := k.Client.CreateMultipartUpload(ctx, &awss3.CreateMultipartUploadInput{
		Bucket: aws.String(k.Region),
		Key:    aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	return &s3.InitiateMultipartUploadResult{
		UploadID: aws.ToString(result.UploadId),
		Bucket:   aws.ToString(result.Bucket),
		Key:      aws.ToString(result.Key),
	}, nil
}

func (k *Kodo) CompleteMultipartUpload(ctx context.Context, uploadID string, name string, parts []s3.Part) (*s3.CompleteMultipartUploadResult, error) {
	kodoParts := make([]awss3types.CompletedPart, len(parts))
	for i, part := range parts {
		kodoParts[i] = awss3types.CompletedPart{
			PartNumber: aws.Int32(int32(part.PartNumber)),
			ETag:       aws.String(part.ETag),
		}
	}
	result, err := k.Client.CompleteMultipartUpload(ctx, &awss3.CompleteMultipartUploadInput{
		Bucket:          aws.String(k.Region),
		Key:             aws.String(name),
		UploadId:        aws.String(uploadID),
		MultipartUpload: &awss3types.CompletedMultipartUpload{Parts: kodoParts},
	})
	if err != nil {
		return nil, err
	}
	return &s3.CompleteMultipartUploadResult{
		Location: aws.ToString(result.Location),
		Bucket:   aws.ToString(result.Bucket),
		Key:      aws.ToString(result.Key),
		ETag:     strings.ToLower(strings.ReplaceAll(aws.ToString(result.ETag), `"`, ``)),
	}, nil
}

func (k *Kodo) PartSize(ctx context.Context, size int64) (int64, error) {
	if size <= 0 {
		return 0, errors.New("size must be greater than 0")
	}
	if size > int64(maxPartSize)*int64(maxNumSize) {
		return 0, fmt.Errorf("size must be less than %db", int64(maxPartSize)*int64(maxNumSize))
	}
	if size <= int64(maxPartSize)*int64(maxNumSize) {
		return minPartSize, nil
	}
	partSize := size / maxNumSize
	if size%maxNumSize != 0 {
		partSize++
	}
	return partSize, nil
}

func (k *Kodo) AuthSign(ctx context.Context, uploadID string, name string, expire time.Duration, partNumbers []int) (*s3.AuthSignResult, error) {
	result := s3.AuthSignResult{
		URL:    k.BucketURL + "/" + name,
		Query:  url.Values{"uploadId": {uploadID}},
		Header: make(http.Header),
		Parts:  make([]s3.SignPart, len(partNumbers)),
	}
	for i, partNumber := range partNumbers {
		part, _ := k.PresignClient.PresignUploadPart(ctx, &awss3.UploadPartInput{
			Bucket:     aws.String(k.Region),
			UploadId:   aws.String(uploadID),
			Key:        aws.String(name),
			PartNumber: aws.Int32(int32(partNumber)),
		})
		result.Parts[i] = s3.SignPart{
			PartNumber: partNumber,
			URL:        part.URL,
			Header:     part.SignedHeader,
		}
	}
	return &result, nil

}

func (k *Kodo) PresignedPutObject(ctx context.Context, name string, expire time.Duration) (string, error) {
	object, err := k.PresignClient.PresignPutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(k.Region),
		Key:    aws.String(name),
	}, awss3.WithPresignExpires(expire), withDisableHTTPPresignerHeaderV4(nil))
	if err != nil {
		return "", err
	}
	return object.URL, nil
}

func (k *Kodo) DeleteObject(ctx context.Context, name string) error {
	_, err := k.Client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(k.Region),
		Key:    aws.String(name),
	})
	return err
}

func (k *Kodo) CopyObject(ctx context.Context, src string, dst string) (*s3.CopyObjectInfo, error) {
	result, err := k.Client.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:     aws.String(k.Region),
		CopySource: aws.String(k.Region + "/" + src),
		Key:        aws.String(dst),
	})
	if err != nil {
		return nil, err
	}
	return &s3.CopyObjectInfo{
		Key:  dst,
		ETag: strings.ToLower(strings.ReplaceAll(aws.ToString(result.CopyObjectResult.ETag), `"`, ``)),
	}, nil
}

func (k *Kodo) StatObject(ctx context.Context, name string) (*s3.ObjectInfo, error) {
	info, err := k.Client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(k.Region),
		Key:    aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	res := &s3.ObjectInfo{Key: name}
	res.Size = aws.ToInt64(info.ContentLength)
	res.ETag = strings.ToLower(strings.ReplaceAll(aws.ToString(info.ETag), `"`, ``))
	return res, nil
}

func (k *Kodo) IsNotFound(err error) bool {
	if err != nil {
		var errorType *awss3types.NotFound
		if errors.As(err, &errorType) {
			return true
		}
	}
	return false
}

func (k *Kodo) AbortMultipartUpload(ctx context.Context, uploadID string, name string) error {
	_, err := k.Client.AbortMultipartUpload(ctx, &awss3.AbortMultipartUploadInput{
		UploadId: aws.String(uploadID),
		Bucket:   aws.String(k.Region),
		Key:      aws.String(name),
	})
	return err
}

func (k *Kodo) ListUploadedParts(ctx context.Context, uploadID string, name string, partNumberMarker int, maxParts int) (*s3.ListUploadedPartsResult, error) {
	result, err := k.Client.ListParts(ctx, &awss3.ListPartsInput{
		Key:              aws.String(name),
		UploadId:         aws.String(uploadID),
		Bucket:           aws.String(k.Region),
		MaxParts:         aws.Int32(int32(maxParts)),
		PartNumberMarker: aws.String(strconv.Itoa(partNumberMarker)),
	})
	if err != nil {
		return nil, err
	}
	res := &s3.ListUploadedPartsResult{
		Key:           aws.ToString(result.Key),
		UploadID:      aws.ToString(result.UploadId),
		MaxParts:      int(aws.ToInt32(result.MaxParts)),
		UploadedParts: make([]s3.UploadedPart, len(result.Parts)),
	}
	// int to string
	NextPartNumberMarker, err := strconv.Atoi(aws.ToString(result.NextPartNumberMarker))
	if err != nil {
		return nil, err
	}
	res.NextPartNumberMarker = NextPartNumberMarker
	for i, part := range result.Parts {
		res.UploadedParts[i] = s3.UploadedPart{
			PartNumber:   int(aws.ToInt32(part.PartNumber)),
			LastModified: aws.ToTime(part.LastModified),
			ETag:         aws.ToString(part.ETag),
			Size:         aws.ToInt64(part.Size),
		}
	}
	return res, nil
}

func (k *Kodo) AccessURL(ctx context.Context, name string, expire time.Duration, opt *s3.AccessURLOption) (string, error) {
	if opt == nil || opt.Image == nil {
		params := &awss3.GetObjectInput{
			Bucket: aws.String(k.Region),
			Key:    aws.String(name),
		}
		res, err := k.PresignClient.PresignGetObject(ctx, params, awss3.WithPresignExpires(expire), withDisableHTTPPresignerHeaderV4(opt))
		if err != nil {
			return "", err
		}
		return res.URL, nil
	}
	//https://developer.qiniu.com/dora/8255/the-zoom
	process := ""
	if opt.Image.Width > 0 {
		process += strconv.Itoa(opt.Image.Width) + "x"
	}
	if opt.Image.Height > 0 {
		if opt.Image.Width > 0 {
			process += strconv.Itoa(opt.Image.Height)
		} else {
			process += "x" + strconv.Itoa(opt.Image.Height)
		}
	}
	imageMogr := "imageMogr2/thumbnail/" + process
	//expire
	deadline := time.Now().Add(expire).Unix()
	domain := k.BucketURL
	query := make(url.Values)
	privateURL := storage.MakePrivateURLv2WithQuery(k.Auth, domain, name, query, deadline)
	if imageMogr != "" {
		privateURL += "&" + imageMogr
	}
	return privateURL, nil
}

func (k *Kodo) SetObjectContentType(ctx context.Context, name string, contentType string) error {
	//set object content-type
	_, err := k.Client.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:            aws.String(k.Region),
		CopySource:        aws.String(k.Region + "/" + name),
		Key:               aws.String(name),
		ContentType:       aws.String(contentType),
		MetadataDirective: awss3types.MetadataDirectiveReplace,
	})
	return err
}
func (k *Kodo) FormData(ctx context.Context, name string, size int64, contentType string, duration time.Duration) (*s3.FormData, error) {
	// https://developer.qiniu.com/kodo/1312/upload
	now := time.Now()
	expiration := now.Add(duration)
	resourceKey := k.Region + ":" + name
	putPolicy := map[string]any{
		"scope":    resourceKey,
		"deadline": now.Unix() + 3600,
	}

	putPolicyJson, err := json.Marshal(putPolicy)
	if err != nil {
		return nil, errs.WrapMsg(err, "Marshal json error")
	}
	encodedPutPolicy := base64.StdEncoding.EncodeToString(putPolicyJson)
	sign := encodedPutPolicy
	h := hmac.New(sha1.New, []byte(k.SecretKey))
	if _, err := io.WriteString(h, sign); err != nil {
		return nil, errs.WrapMsg(err, "WriteString error")
	}

	encodedSign := base64.StdEncoding.EncodeToString([]byte(sign))
	uploadToken := k.AccessKey + ":" + encodedSign + ":" + encodedPutPolicy

	fd := &s3.FormData{
		URL:     k.BucketURL,
		File:    "file",
		Expires: expiration,
		FormData: map[string]string{
			"key":   resourceKey,
			"token": uploadToken,
		},
		SuccessCodes: []int{successCode},
	}
	if contentType != "" {
		fd.FormData["accept"] = contentType
	}
	return fd, nil
}

func withDisableHTTPPresignerHeaderV4(opt *s3.AccessURLOption) func(options *awss3.PresignOptions) {
	return func(options *awss3.PresignOptions) {
		options.Presigner = &disableHTTPPresignerHeaderV4{
			opt:       opt,
			presigner: options.Presigner,
		}
	}
}

type disableHTTPPresignerHeaderV4 struct {
	opt       *s3.AccessURLOption
	presigner awss3.HTTPPresignerV4
}

func (d *disableHTTPPresignerHeaderV4) PresignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash string, service string, region string, signingTime time.Time, optFns ...func(*v4.SignerOptions)) (url string, signedHeader http.Header, err error) {
	optFns = append(optFns, func(options *v4.SignerOptions) {
		options.DisableHeaderHoisting = true
	})
	r.Header.Del("Amz-Sdk-Request")
	d.setOption(r.URL)
	return d.presigner.PresignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
}

func (d *disableHTTPPresignerHeaderV4) setOption(u *url.URL) {
	if d.opt == nil {
		return
	}
	query := u.Query()
	if d.opt.ContentType != "" {
		query.Set("response-content-type", d.opt.ContentType)
	}
	if d.opt.Filename != "" {
		query.Set("response-content-disposition", `attachment; filename*=UTF-8''`+url.PathEscape(d.opt.Filename))
	}
	u.RawQuery = query.Encode()
}
