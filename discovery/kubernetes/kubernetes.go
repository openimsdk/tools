package kubernetes

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	"github.com/sercand/kuberesolver/v6"
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

type addrConn struct {
	conn        *grpc.ClientConn
	addr        string
	isConnected bool
}

type ConnManager struct {
	clientset   *kubernetes.Clientset
	namespace   string
	dialOptions []grpc.DialOption

	selfTarget string

	// watchNames denotes the service names for which notifications may need to be sent to all nodes (pods),
	// and maintains a separate list of direct connections to pods that bypass the Service.
	watchNames []string
	connsMu    sync.RWMutex
	connsMap   map[string][]*addrConn
}

// NewConnManager creates a new connection manager that uses Kubernetes services for service discovery.
func NewConnManager(namespace string, watchNames []string, options ...grpc.DialOption) (*ConnManager, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to create in-cluster config:")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to create clientset:")

	}

	kuberesolver.RegisterInCluster()

	k := &ConnManager{
		clientset:   clientset,
		namespace:   namespace,
		dialOptions: options,
		watchNames:  watchNames,
		connsMap:    make(map[string][]*addrConn),
	}

	go k.watchEndpoints()

	return k, nil
}

func (k *ConnManager) buildTarget(serviceName string, svcPort int32) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", serviceName, k.namespace, svcPort)
}

func (k *ConnManager) buildAddr(serviceName string) string {
	return "kubernetes:///" + serviceName
}

func (k *ConnManager) initializeConns(serviceName string, opts ...grpc.DialOption) error {
	k.connsMu.Lock()
	defer k.connsMu.Unlock()

	// 1. take a snapshot of old connections
	oldList := k.connsMap[serviceName]
	addrMap := make(map[string]*addrConn, len(oldList))
	for _, ac := range oldList {
		addrMap[ac.addr] = ac
	}

	// 2. fetch all Endpoints for this Service
	eps, err := k.clientset.CoreV1().
		Endpoints(k.namespace).
		Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		return errs.WrapMsg(err, "failed to get endpoints", "serviceName", serviceName)
	}

	// 3. rebuild connection list
	var newList []*addrConn
	for _, subset := range eps.Subsets {
		for _, address := range subset.Addresses {
			for _, port := range subset.Ports {
				if port.Name != GRPCName {
					continue
				}
				addr := fmt.Sprintf("%s:%d", address.IP, port.Port)

				// reuse existing if present
				if ac, ok := addrMap[addr]; ok {
					ac.isConnected = true
					continue
				}

				// dial a brand-new connection
				dialOpts := append(append(k.dialOptions, opts...), grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err := k.checkOpts(dialOpts...); err != nil {
					return errs.WrapMsg(err, "checkOpts failed", "addr", addr)
				}
				conn, err := grpc.DialContext(context.Background(), addr, dialOpts...)
				if err != nil {
					// skip unreachable endpoints
					continue
				}
				newList = append(newList, &addrConn{conn: conn, addr: addr, isConnected: false})
			}
		}
	}

	// 4. close any old connections that weren’t reused
	for _, conn := range oldList {
		if conn.isConnected {
			conn.isConnected = false
			newList = append(newList, conn)
			continue
		}
		if err = conn.conn.Close(); err != nil {
			log.ZWarn(context.TODO(), "close conn err", err)
		}
	}

	// 5. replace map entry
	k.connsMap[serviceName] = newList
	return nil
}

// GetConns returns gRPC client connections for a given Kubernetes service name.
func (k *ConnManager) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error) {
	k.connsMu.RLock()
	if len(k.connsMap) == 0 {
		k.connsMu.RUnlock()
		if err := k.initializeConns(serviceName, opts...); err != nil {
			return nil, err
		}
		k.connsMu.RLock()
	}
	defer k.connsMu.RUnlock()

	return datautil.Batch(func(t *addrConn) grpc.ClientConnInterface { return t.conn }, k.connsMap[serviceName]), nil

}

// GetConn returns a single gRPC client connection for a given Kubernetes service name.
func (k *ConnManager) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error) {
	// In Kubernetes, we can directly use the service name for service discovery
	// Using headless service approach - just serviceName without getting port
	target := k.buildAddr(serviceName)

	dialOpts := append(append(k.dialOptions, opts...),
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	err := k.checkOpts(dialOpts...) // Check opts in include mw.GrpcClient()
	if err != nil {
		return nil, errs.WrapMsg(err, "checkOpts is failed")
	}

	return grpc.DialContext(ctx, target, dialOpts...)
}

// GetSelfConnTarget returns the connection target for the current service.
func (k *ConnManager) GetSelfConnTarget() string {
	if k.selfTarget == "" {
		ctx := context.TODO()
		hostName := os.Getenv("HOSTNAME")

		pod, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, hostName, metav1.GetOptions{})
		if err != nil {
			log.ZWarn(ctx, "failed to get pod", err, "selfTarget", hostName)
		}

		for i := 0; i < 5; i++ {
			pod, err = k.clientset.CoreV1().Pods(k.namespace).Get(ctx, hostName, metav1.GetOptions{})
			if err == nil {
				break
			}

			time.Sleep(3 * time.Second)
		}
		if err != nil {
			log.ZWarn(ctx, "Error getting pod", err, "hostName", hostName)
			return ""
		}

		var (
			selfPort, elsePort int32
		)

		log.ZDebug(ctx, "getSelfPods containers length", len(pod.Spec.Containers), "hostname", hostName)

		for _, port := range pod.Spec.Containers[0].Ports {
			if port.Name == GRPCName {
				selfPort = port.ContainerPort
				break
			} else {
				elsePort = port.ContainerPort
			}
			log.ZDebug(ctx, "getSelfPods port", port.ContainerPort)
		}
		if selfPort == 0 {
			selfPort = elsePort
		}

		k.selfTarget = fmt.Sprintf("%s:%d", pod.Status.PodIP, selfPort)
		log.ZDebug(ctx, "getSelfPods selfTarget", k.selfTarget)
	}

	return k.selfTarget
}

func (k *ConnManager) IsSelfNode(cc grpc.ClientConnInterface) bool {
	cli, ok := cc.(*grpc.ClientConn)
	ctx := context.TODO()
	log.ZDebug(ctx, "isSelfNode ok", ok)
	if !ok {
		return false
	}
	log.ZDebug(ctx, "isSelfNode target", cli.Target())
	return k.GetSelfConnTarget() == cli.Target()
}

// AddOption appends gRPC dial options to the existing options.
func (k *ConnManager) AddOption(opts ...grpc.DialOption) {
	k.connsMu.Lock()
	defer k.connsMu.Unlock()
	k.resetConnMap()
	k.dialOptions = append(k.dialOptions, opts...)
}

// CloseConn closes a given gRPC client connection.
//func (k *ConnManager) CloseConn(conn *grpc.ClientConn) {
//	conn.Close()
//}

// Close closes all gRPC connections managed by ConnManager.
func (k *ConnManager) Close() {
	k.connsMu.Lock()
	defer k.connsMu.Unlock()
	k.resetConnMap()
}

func (k *ConnManager) Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error {
	return nil
}

func (k *ConnManager) UnRegister() error {
	return nil
}

func (k *ConnManager) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	return "", nil
}

func (k *ConnManager) getServicePort(serviceName string) (int32, error) {
	var svcPort int32

	svc, err := k.clientset.CoreV1().Services(k.namespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.ZWarn(context.Background(), "service not found", err, "serviceName", serviceName)
			return 0, nil
		}
		return 0, errs.WrapMsg(err, "failed to get service", "serviceName", serviceName)
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == GRPCName {
			svcPort = port.Port
			break
		}
	}

	return svcPort, nil
}

// watchEndpoints listens for changes in Endpoints resources.
func (k *ConnManager) watchEndpoints() {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(k.clientset, time.Minute*10,
		informers.WithNamespace(k.namespace))
	informer := informerFactory.Core().V1().Endpoints().Informer()

	// Watch for Endpoints changes (add, update, delete)
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

func (k *ConnManager) handleEndpointChange(obj interface{}) {
	endpoint, ok := obj.(*v1.Endpoints)
	if !ok {
		return
	}
	serviceName := endpoint.Name
	if datautil.Contain(serviceName, k.watchNames...) {
		if err := k.initializeConns(serviceName); err != nil {
			log.ZWarn(context.Background(), "Error initializing connections", err, "serviceName", serviceName)
		}
	}
}

func (k *ConnManager) checkOpts(opts ...grpc.DialOption) error {
	// mwOpt := mw.GrpcClient()

	// for _, opt := range opts {
	// 	if opt == mwOpt {
	// 		return nil
	// 	}
	// }

	// return errs.New("missing required grpc.DialOption", "option", "mw.GrpcClient")

	return nil
}

func (k *ConnManager) SetKey(ctx context.Context, key string, data []byte) error {
	return discovery.ErrNotSupported
}

func (k *ConnManager) SetWithLease(ctx context.Context, key string, val []byte, ttl int64) error {
	return discovery.ErrNotSupported
}

func (k *ConnManager) GetKey(ctx context.Context, key string) ([]byte, error) {
	return nil, discovery.ErrNotSupported
}

func (k *ConnManager) GetKeyWithPrefix(ctx context.Context, key string) ([][]byte, error) {
	return nil, discovery.ErrNotSupported
}

func (k *ConnManager) DelData(ctx context.Context, key string) error {
	return discovery.ErrNotSupported
}

func (k *ConnManager) WatchKey(ctx context.Context, key string, fn discovery.WatchKeyHandler) error {
	return discovery.ErrNotSupported
}

func (k *ConnManager) resetConnMap() {
	ctx := context.Background()
	for _, conn := range k.connsMap {
		for _, c := range conn {
			if err := c.conn.Close(); err != nil {
				log.ZWarn(ctx, "failed to close conn", err)
			}
		}
	}
	k.connsMap = make(map[string][]*addrConn)
}
