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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHTTPClient_Get(t *testing.T) {
	// Setup a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock response"))
	}))
	defer server.Close()

	client := NewHTTPClient(NewClientConfig())
	body, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(body) != "mock response" {
		t.Fatalf("Expected 'mock response', got %s", body)
	}
}

func TestHTTPClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST method, got %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		w.Write(body)
	}))
	defer server.Close()

	client := NewHTTPClient(NewClientConfig())
	headers := map[string]string{"Custom-Header": "value"}

	expectedData := map[string]string{"key": "value"}
	respBody, err := client.Post(context.Background(), server.URL, headers, expectedData, 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var expected, actual map[string]any
	if err := json.Unmarshal([]byte(`{"key":"value"}`), &expected); err != nil {
		t.Fatalf("Error unmarshaling expected JSON: %v", err)
	}
	if err := json.Unmarshal(respBody, &actual); err != nil {
		t.Fatalf("Error unmarshaling actual response body: %v", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func TestHTTPClient_PostReturn(t *testing.T) {
	expectedOutput := struct{ Key string }{"value"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(expectedOutput)
	}))
	defer server.Close()

	client := NewHTTPClient(NewClientConfig())
	var actualOutput struct{ Key string }
	err := client.PostReturn(context.Background(), server.URL, nil, map[string]string{"key": "value"}, &actualOutput, 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if actualOutput.Key != expectedOutput.Key {
		t.Fatalf("Expected %s, got %s", expectedOutput.Key, actualOutput.Key)
	}
}
