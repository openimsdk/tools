// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package minio

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/s3"
)

func (m *Minio) getImageThumbnailURL(ctx context.Context, name string, expire time.Duration, opt *s3.Image) (string, error) {
	var img image.Image
	info, err := m.cache.GetImageObjectKeyInfo(ctx, name, func(ctx context.Context) (info *ImageInfo, err error) {
		info, img, err = m.getObjectImageInfo(ctx, name)
		return
	})
	if err != nil {
		return "", err
	}
	if !info.IsImg {
		return "", errs.New("object not image").Wrap()
	}
	if opt.Width > info.Width || opt.Width <= 0 {
		opt.Width = info.Width
	}
	if opt.Height > info.Height || opt.Height <= 0 {
		opt.Height = info.Height
	}
	opt.Format = strings.ToLower(opt.Format)
	if opt.Format == formatJpg {
		opt.Format = formatJpeg
	}
	switch opt.Format {
	case formatPng, formatJpeg, formatGif:
	default:
		opt.Format = formatPng
	}
	reqParams := make(url.Values)
	if opt.Width == info.Width && opt.Height == info.Height && (opt.Format == info.Format || opt.Format == "") {
		reqParams.Set("response-content-type", "image/"+info.Format)
		return m.PresignedGetObject(ctx, name, expire, reqParams)
	}
	key, err := m.cache.GetThumbnailKey(ctx, name, opt.Format, opt.Width, opt.Height, func(ctx context.Context) (string, error) {
		if img == nil {
			var reader *minio.Object
			reader, err = m.core.Client.GetObject(ctx, m.bucket, name, minio.GetObjectOptions{})
			if err != nil {
				return "", err
			}
			defer reader.Close()
			img, _, err = ImageStat(reader)
			if err != nil {
				return "", err
			}
		}
		thumbnail := resizeImage(img, opt.Width, opt.Height)
		buf := bytes.NewBuffer(nil)
		switch opt.Format {
		case formatPng:
			err = png.Encode(buf, thumbnail)
		case formatJpeg:
			err = jpeg.Encode(buf, thumbnail, &jpeg.Options{Quality: 40})
		case formatGif:
			err = gif.Encode(buf, thumbnail, nil)
		}
		if err != nil {
			return "", errs.WrapMsg(err, "encode failed", "type", opt.Format)
		}
		cacheKey := filepath.Join(imageThumbnailPath, info.Etag, fmt.Sprintf("image_w%d_h%d.%s", opt.Width, opt.Height, opt.Format))
		if _, err = m.core.Client.PutObject(ctx, m.bucket, cacheKey, buf, int64(buf.Len()), minio.PutObjectOptions{}); err != nil {
			return "", err
		}
		return cacheKey, nil
	})
	if err != nil {
		return "", err
	}
	reqParams.Set("response-content-type", "image/"+opt.Format)
	return m.PresignedGetObject(ctx, key, expire, reqParams)
}

func (m *Minio) getObjectImageInfo(ctx context.Context, name string) (*ImageInfo, image.Image, error) {
	fileInfo, err := m.StatObject(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	if fileInfo.Size > maxImageSize {
		return nil, nil, errs.New("file size too large").Wrap()
	}
	imageData, err := m.getObjectData(ctx, name, fileInfo.Size)
	if err != nil {
		return nil, nil, err
	}
	var info ImageInfo
	imageInfo, format, err := ImageStat(bytes.NewReader(imageData))
	if err == nil {
		info.IsImg = true
		info.Format = format
		info.Width, info.Height = ImageWidthHeight(imageInfo)
	} else {
		info.IsImg = false
	}
	info.Etag = fileInfo.ETag
	return &info, imageInfo, nil
}

func (m *Minio) delObjectImageInfoKey(ctx context.Context, key string, size int64) {
	if size > 0 && size > maxImageSize {
		return
	}
	if err := m.cache.DelObjectImageInfoKey(ctx, key); err != nil {
		log.ZError(ctx, "DelObjectImageInfoKey failed", err, "key", key)
	}
}
