// GCS 实现 s3.Interface
package gcp

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openimsdk/tools/s3"
)

var testClient *GCS

func initTestClient(t *testing.T) *GCS {
	if testClient != nil {
		return testClient
	}
	cfgJson, err := os.ReadFile("E:\\im\\im.json")
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	conf := Config{
		Bucket:         "uni-resource",
		CredentialJSON: cfgJson,
	}
	client, err := NewGCS(conf)
	if err != nil {
		t.Fatalf("init client failed: %v", err)
	}
	testClient = client
	return client
}

func TestPresignedPut(t *testing.T) {
	client := initTestClient(t)
	key := "gcp-test.txt"
	urlRes, err := client.PresignedPutObject(context.Background(), key, 10*time.Minute, &s3.PutOption{ContentType: "text/plain"})
	if err != nil {
		t.Fatalf("presign failed: %v", err)
	}

	req, _ := http.NewRequest("PUT", urlRes.URL, strings.NewReader("hello from GCS test"))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("upload failed: %v", err)
	}
}

func TestAccessURL(t *testing.T) {
	client := initTestClient(t)
	key := "gcp-test.txt"
	downloadURL, err := client.AccessURL(context.Background(), key, 5*time.Minute, &s3.AccessURLOption{
		Filename:    "download.txt",
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("access URL failed: %v", err)
	}
	t.Log("Download URL:", downloadURL)

	resp, err := http.Get(downloadURL)
	if err != nil {
		t.Fatalf("download request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected download status: %v", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	t.Logf("Downloaded content: %s", string(body))
}

func TestFormDataUpload(t *testing.T) {
	client := initTestClient(t)
	key := "gcp-test.txt"
	form, err := client.FormData(context.Background(), key, 1024*1024, "text/plain", 10*time.Minute)
	if err != nil {
		t.Fatalf("form policy failed: %v", err)
	}

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	for k, v := range form.FormData {
		_ = writer.WriteField(k, v)
	}
	fw, _ := writer.CreateFormFile(form.File, key)
	_, _ = fw.Write([]byte("form upload data test"))
	writer.Close()

	req, _ := http.NewRequest("POST", form.URL, buf)
	req.Header = form.Header
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 204 && resp.StatusCode != 201) {
		t.Fatalf("form upload failed: %v", err)
	}
}

func TestStatObject(t *testing.T) {
	client := initTestClient(t)
	key := "gcp-test.txt"
	info, err := client.StatObject(context.Background(), key)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	t.Logf("Stat: key=%s, size=%d", info.Key, info.Size)
}

func TestDeleteObject(t *testing.T) {
	client := initTestClient(t)
	err := client.DeleteObject(context.Background(), "gcp-test.txt")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestBatchAll(t *testing.T) {
	t.Run("PresignedPut", TestPresignedPut)
	t.Run("AccessURL", TestAccessURL)
	t.Run("FormDataUpload", TestFormDataUpload)
	t.Run("StatObject", TestStatObject)
	t.Run("DeleteObject", TestDeleteObject)
}
