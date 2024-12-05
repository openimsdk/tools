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
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesConnManager struct {
	clientset   *kubernetes.Clientset
	namespace   string
	dialOptions []grpc.DialOption

	selfTarget string

	mu      sync.RWMutex
	connMap map[string][]*grpc.ClientConn
}

// NewKubernetesConnManager creates a new connection manager that uses Kubernetes services for service discovery.
func NewKubernetesConnManager(namespace string, options ...grpc.DialOption) (*KubernetesConnManager, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return &KubernetesConnManager{
		clientset:   clientset,
		namespace:   namespace,
		dialOptions: options,
		connMap:     make(map[string][]*grpc.ClientConn),
	}, nil
}

func (k *KubernetesConnManager) initializeConnMap() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	services, err := k.clientset.CoreV1().Services(k.namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	for _, service := range services.Items {
		endpoints, err := k.clientset.CoreV1().Endpoints(k.namespace).Get(context.Background(), service.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get endpoints for service %s: %v", service.Name, err)
		}

		var conns []*grpc.ClientConn
		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				conn, err := grpc.Dial(address.IP, append(k.dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))...)
				if err != nil {
					return fmt.Errorf("failed to dial endpoint %s: %v", address.IP, err)
				}
				conns = append(conns, conn)
			}
		}
		k.connMap[service.Name] = conns
	}

	return nil
}

// GetConns returns gRPC client connections for a given Kubernetes service name.
func (k *KubernetesConnManager) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]*grpc.ClientConn, error) {
	// k.mu.RLock()
	// defer k.mu.RUnlock()
	// if len(k.connMap) == 0 {
	// 	if err := k.initializeConnMap(); err != nil {
	// 		return nil, err
	// 	}
	// }

	// return k.connMap[serviceName], nil
	return nil, nil
}

// GetConn returns a single gRPC client connection for a given Kubernetes service name.
func (k *KubernetesConnManager) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	// k.mu.RLock()
	// if len(k.connMap) == 0 {
	// 	k.mu.RUnlock()
	// 	if err := k.initializeConnMap(); err != nil {
	// 		return nil, err
	// 	}
	// 	k.mu.RLock()
	// }

	// defer k.mu.RUnlock()

	// return k.connMap[serviceName][0], nil
	return grpc.DialContext(ctx, serviceName, append(k.dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))...)
}

// GetSelfConnTarget returns the connection target for the current service.
func (k *KubernetesConnManager) GetSelfConnTarget() string {
	return k.selfTarget
}

// AddOption appends gRPC dial options to the existing options.
func (k *KubernetesConnManager) AddOption(opts ...grpc.DialOption) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.dialOptions = append(k.dialOptions, opts...)
}

// CloseConn closes a given gRPC client connection.
func (k *KubernetesConnManager) CloseConn(conn *grpc.ClientConn) {
	conn.Close()
}

// Close closes all gRPC connections managed by KubernetesConnManager.
func (k *KubernetesConnManager) Close() {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, conns := range k.connMap {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}
	k.connMap = make(map[string][]*grpc.ClientConn)
}

func (k *KubernetesConnManager) Register(serviceName, host string, port int, opts ...grpc.DialOption) error {
	return nil
}
func (k *KubernetesConnManager) UnRegister() error {
	return nil
}

func (k *KubernetesConnManager) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	return "", nil
}
