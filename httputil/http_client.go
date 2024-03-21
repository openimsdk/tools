package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/openimsdk/tools/errs"
)

// ClientConfig defines configuration for the HTTP client.
type ClientConfig struct {
	Timeout         time.Duration
	MaxConnsPerHost int
}

// NewClientConfig creates a default client configuration.
func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:         15 * time.Second,
		MaxConnsPerHost: 100,
	}
}

// HTTPClient wraps http.Client and includes additional configuration.
type HTTPClient struct {
	client *http.Client
	config *ClientConfig
}

// NewHTTPClient creates a new HTTPClient with the provided configuration.
func NewHTTPClient(config *ClientConfig) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxConnsPerHost: config.MaxConnsPerHost,
			},
		},
		config: config,
	}
}

// Get performs a HTTP GET request and returns the response body.
func (c *HTTPClient) Get(url string) ([]byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, errs.WrapMsg(err, "GET request failed", "url", url)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to read response body", "url", url)
	}
	return body, nil
}

// Post sends a JSON-encoded POST request and returns the response body.
func (c *HTTPClient) Post(ctx context.Context, url string, headers map[string]string, data any) ([]byte, error) {
	body := bytes.NewBuffer(nil)
	if data != nil {
		if err := json.NewEncoder(body).Encode(data); err != nil {
			return nil, errs.WrapMsg(err, "JSON encode failed", "data", data)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, errs.WrapMsg(err, "NewRequestWithContext failed", "url", url, "method", http.MethodPost)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errs.WrapMsg(err, "HTTP request failed")
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to read response body")
	}

	return result, nil
}

// PostReturn sends a JSON-encoded POST request and decodes the JSON response into the output parameter.
func (c *HTTPClient) PostReturn(ctx context.Context, url string, headers map[string]string, input, output any) error {
	responseBytes, err := c.Post(ctx, url, headers, input)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(responseBytes, output); err != nil {
		return errs.WrapMsg(err, "JSON unmarshal failed")
	}
	return nil
}
