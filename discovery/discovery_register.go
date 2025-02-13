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

package discovery

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

var ErrNotSupportedKeyValue = errors.New("discovery data not supported key value")

type Conn interface {
	GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error)
	GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error)
	IsSelfNode(cc grpc.ClientConnInterface) bool
}

type WatchKey struct {
	Value []byte
}

type WatchKeyHandler func(data *WatchKey) error

type KeyValue interface {
	SetKey(ctx context.Context, key string, value []byte) error
	GetKey(ctx context.Context, key string) ([]byte, error)
	WatchKey(ctx context.Context, key string, fn WatchKeyHandler) error
}

type SvcDiscoveryRegistry interface {
	Conn
	KeyValue
	AddOption(opts ...grpc.DialOption)
	Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error
	Close()
	GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error)
}
