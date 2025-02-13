package minio

import (
	"context"
	"path"
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	conf := Config{
		Bucket:          "openim",
		AccessKeyID:     "root",
		SecretAccessKey: "openIM123",
		Endpoint:        "http://127.0.0.1:10005",
	}
	ctx := context.Background()
	m, err := NewMinio(ctx, nil, conf)
	if err != nil {
		panic(err)
	}
	t.Log(m.DeleteObject(ctx, "/openim/data/hash/6aeb6959cad0d0b2ef4a5d9f66ed394a"))
	//t.Log(m.deleteThumbnailCache(ctx, "/openim/data/hash/6aeb6959cad0d0b2ef4a5d9f66ed394a", "openim/thumbnail/ae20fe3d6466fdb11bcf465386b51312/image_w640_h640.jpeg"))
	//p := path.Join(imageThumbnailPath, "039395be6547fb10724fd0999ea3e834", "/")
	//res := m.core.Client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{Prefix: p})
	//for re := range res {
	//	t.Log(re)
	//}
}

func TestName2(t *testing.T) {
	t.Log(strings.Trim(path.Base("openim/thumbnail/ae20fe3d6466fdb11bcf465386b51312/image_w640_h640.jpeg"), "."))

}
