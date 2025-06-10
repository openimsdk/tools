package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	GRPCName = "grpc"
)

type KubernetesConnManager struct {
	clientset   *kubernetes.Clientset
	namespace   string
	dialOptions []grpc.DialOption

	rpcTargets map[string]string
	selfTarget string

	mu      sync.RWMutex
	connMap map[string][]grpc.ClientConnInterface
}

// NewKubernetesConnManager creates a new connection manager that uses Kubernetes services for service discovery.
func NewKubernetesConnManager(namespace string, options ...grpc.DialOption) (*KubernetesConnManager, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to create in-cluster config:")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to create clientset:")

	}

	k := &KubernetesConnManager{
		clientset:   clientset,
		namespace:   namespace,
		dialOptions: options,
		connMap:     make(map[string][]grpc.ClientConnInterface),
	}

	go k.watchEndpoints()

	return k, nil
}

func (k *KubernetesConnManager) initializeConns(serviceName string, opts ...grpc.DialOption) error {
	port, err := k.getServicePort(serviceName)
	if err != nil {
		return errs.Wrap(err)
	}
	endpoints, err := k.clientset.CoreV1().Endpoints(k.namespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		return errs.WrapMsg(err, "failed to get endpoints", "serviceName", serviceName)
	}

	// fmt.Println("Endpoints:", endpoints, "endpoints.Subsets:", endpoints.Subsets)

	var conns []grpc.ClientConnInterface
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			target := fmt.Sprintf("%s:%d", address.IP, port)
			// fmt.Println("IP target:", target)

			dialOpts := append(append(k.dialOptions, opts...),
				grpc.WithTransportCredentials(insecure.NewCredentials()))

			err := k.checkOpts(dialOpts...) // Check opts in include mw.GrpcClient()
			if err != nil {
				return errs.WrapMsg(err, "checkOpts is failed")
			}

			conn, err := grpc.DialContext(context.Background(), target,
				dialOpts...)
			if err != nil {
				return errs.WrapMsg(err, "failed to dial endpoint", "target", target)
			}
			conns = append(conns, conn)
		}
	}

	k.mu.Lock()
	k.connMap[serviceName] = conns
	k.mu.Unlock()

	return nil
}

// GetConns returns gRPC client connections for a given Kubernetes service name.
func (k *KubernetesConnManager) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error) {
	k.mu.RLock()

	conns, exists := k.connMap[serviceName]
	k.mu.RUnlock()
	if exists {
		return conns, nil
	}

	k.mu.Lock()
	// Check if another goroutine has already initialized the connections when we released the read lock
	conns, exists = k.connMap[serviceName]
	if exists {
		return conns, nil
	}
	k.mu.Unlock()

	if err := k.initializeConns(serviceName, opts...); err != nil {

		return nil, errs.WrapMsg(err, "Failed to initialize connections for service", "serviceName", serviceName)
	}

	return k.connMap[serviceName], nil
}

// GetConn returns a single gRPC client connection for a given Kubernetes service name.
func (k *KubernetesConnManager) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error) {
	var target string

	if k.rpcTargets[serviceName] == "" {
		var err error

		svcPort, err := k.getServicePort(serviceName)
		if err != nil {
			return nil, err
		}

		target = fmt.Sprintf("%s.%s.svc.cluster.local:%d", serviceName, k.namespace, svcPort)

		// fmt.Println("SVC target:", target)
	} else {
		target = k.rpcTargets[serviceName]
	}

	dialOpts := append(append(k.dialOptions, opts...),
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	err := k.checkOpts(dialOpts...) // Check opts in include mw.GrpcClient()
	if err != nil {
		return nil, errs.WrapMsg(err, "checkOpts is failed")
	}

	return grpc.DialContext(ctx, target, dialOpts...)
}

// GetSelfConnTarget returns the connection target for the current service.
func (k *KubernetesConnManager) GetSelfConnTarget() string {
	if k.selfTarget == "" {
		ctx := context.TODO()
		hostName := os.Getenv("HOSTNAME")

		pod, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, hostName, metav1.GetOptions{})
		if err != nil {
			log.ZWarn(ctx, "failed to get pod", err, "selfTarget", hostName)
		}

		for pod.Status.PodIP == "" {
			pod, err = k.clientset.CoreV1().Pods(k.namespace).Get(ctx, hostName, metav1.GetOptions{})
			if err != nil {
				log.ZWarn(ctx, "Error getting pod", err)
			}

			time.Sleep(3 * time.Second)
		}

		var selfPort int32

		for _, port := range pod.Spec.Containers[0].Ports {
			if port.Name == GRPCName {
				selfPort = port.ContainerPort
				break
			}
		}

		k.selfTarget = fmt.Sprintf("%s:%d", pod.Status.PodIP, selfPort)
	}

	return k.selfTarget
}

func (k *KubernetesConnManager) IsSelfNode(cc grpc.ClientConnInterface) bool {
	cli, ok := cc.(*grpc.ClientConn)
	if !ok {
		return false
	}
	return k.GetSelfConnTarget() == cli.Target()
}

// AddOption appends gRPC dial options to the existing options.
func (k *KubernetesConnManager) AddOption(opts ...grpc.DialOption) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.dialOptions = append(k.dialOptions, opts...)
}

// CloseConn closes a given gRPC client connection.
//func (k *KubernetesConnManager) CloseConn(conn *grpc.ClientConn) {
//	conn.Close()
//}

// Close closes all gRPC connections managed by KubernetesConnManager.
func (k *KubernetesConnManager) Close() {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, conns := range k.connMap {
		for _, conn := range conns {
			if closer, ok := conn.(io.Closer); ok {
				_ = closer.Close()
			}
		}
	}
	k.connMap = make(map[string][]grpc.ClientConnInterface)
}

func (k *KubernetesConnManager) Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error {
	return nil
}

func (k *KubernetesConnManager) UnRegister() error {
	return nil
}

func (k *KubernetesConnManager) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	return "", nil
}

func (k *KubernetesConnManager) getServicePort(serviceName string) (int32, error) {
	var svcPort int32

	svc, err := k.clientset.CoreV1().Services(k.namespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.ZWarn(context.Background(), "service not found", err, "serviceName", serviceName)
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get service %s: %v", serviceName, err)
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == GRPCName {
			svcPort = port.Port
			break
		}
	}

	return svcPort, nil
}

// watchEndpoints listens for changes in Pod resources.
func (k *KubernetesConnManager) watchEndpoints() {
	informerFactory := informers.NewSharedInformerFactory(k.clientset, time.Minute*10)
	informer := informerFactory.Core().V1().Pods().Informer()

	// Watch for Pod changes (add, update, delete)
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			k.handleEndpointChange(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			k.handleEndpointChange(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			k.handleEndpointChange(obj)
		},
	})

	informerFactory.Start(context.Background().Done())
	<-context.Background().Done() // Block forever
}

func (k *KubernetesConnManager) handleEndpointChange(obj interface{}) {
	endpoint, ok := obj.(*v1.Endpoints)
	if !ok {
		return
	}
	serviceName := endpoint.Name
	if err := k.initializeConns(serviceName); err != nil {
		log.ZWarn(context.Background(), "Error initializing connections", err, "serviceName", serviceName)
	}
}

func (k *KubernetesConnManager) checkOpts(opts ...grpc.DialOption) error {
	// mwOpt := mw.GrpcClient()

	// for _, opt := range opts {
	// 	if opt == mwOpt {
	// 		return nil
	// 	}
	// }

	// return errs.New("missing required grpc.DialOption", "option", "mw.GrpcClient")

	return nil
}

func (k *KubernetesConnManager) SetKey(ctx context.Context, key string, data []byte) error {
	return discovery.ErrNotSupported
}

func (k *KubernetesConnManager) SetWithLease(ctx context.Context, key string, val []byte, ttl int64) error {
	return discovery.ErrNotSupported
}

func (k *KubernetesConnManager) GetKey(ctx context.Context, key string) ([]byte, error) {
	return nil, discovery.ErrNotSupported
}

func (k *KubernetesConnManager) GetKeyWithPrefix(ctx context.Context, key string) ([][]byte, error) {
	return nil, discovery.ErrNotSupported
}

func (k *KubernetesConnManager) DelData(ctx context.Context, key string) error {
	return discovery.ErrNotSupported
}

func (k *KubernetesConnManager) WatchKey(ctx context.Context, key string, fn discovery.WatchKeyHandler) error {
	return discovery.ErrNotSupported
}
