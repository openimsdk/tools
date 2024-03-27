// Copyright © 2024 OpenIM open source community. All rights reserved.
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
func (c *HTTPClient) Post(ctx context.Context, url string, headers map[string]string, data any, timeout int) ([]byte, error) {
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(timeout))
		defer cancel()
	}
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

// PostReturn sends a JSON-encoded POST request and decodes the JSON response into  output parameter.
func (c *HTTPClient) PostReturn(ctx context.Context, url string, headers map[string]string, input, output any, timeout int) error {
	responseBytes, err := c.Post(ctx, url, headers, input, timeout)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(responseBytes, output); err != nil {
		return errs.WrapMsg(err, "JSON unmarshal failed")
	}
	return nil
}
