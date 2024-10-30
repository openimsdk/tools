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

package kubernetes

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sync"
)

type K8sConnManager struct {
	clientset   *kubernetes.Clientset
	namespace   string
	dialOptions []grpc.DialOption

	selfTarget string

	mu      sync.RWMutex
	connMap map[string][]*grpc.ClientConn
}

// NewK8sConnManager creates a new connection manager that uses Kubernetes services for service discovery
func NewK8sConnManager(namespace string, options ...grpc.DialOption) (*K8sConnManager, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return &K8sConnManager{
		clientset:   clientset,
		namespace:   namespace,
		dialOptions: options,
		connMap:     make(map[string][]*grpc.ClientConn),
	}, nil
}

// GetConns returns gRPC client connections for a given Kubernetes service name
func (k *K8sConnManager) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]*grpc.ClientConn, error) {
	return nil, nil
}

// GetConn returns a single gRPC client connection for a given Kubernetes service name
func (k *K8sConnManager) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, serviceName, append(k.dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))...)
}

// GetSelfConnTarget returns the connection target for the current service
func (k *K8sConnManager) GetSelfConnTarget() string {
	return k.selfTarget
}

// AddOption appends gRPC dial options to the existing options
func (k *K8sConnManager) AddOption(opts ...grpc.DialOption) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.dialOptions = append(k.dialOptions, opts...)
}

// CloseConn closes a given gRPC client connection
func (k *K8sConnManager) CloseConn(conn *grpc.ClientConn) {
	_ = conn.Close()
}

// Close closes all gRPC connections managed by K8sConnManager
func (k *K8sConnManager) Close() {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, conns := range k.connMap {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}
	k.connMap = make(map[string][]*grpc.ClientConn)
}

func (k *K8sConnManager) Register(serviceName, host string, port int, opts ...grpc.DialOption) error {
	return nil
}
func (k *K8sConnManager) UnRegister() error {
	return nil
}

func (k *K8sConnManager) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	return "", nil
}
