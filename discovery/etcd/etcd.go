package etcd

import (
	"context"
	"io"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	gresolver "google.golang.org/grpc/resolver"
)

const (
	defaultRegisterTimeout = 5 * time.Second
	defaultLeaseTTL        = int64(30)
	keepAliveRetryDelay    = time.Second
	defaultCloseTimeout    = 5 * time.Second
)

// CfgOption defines a function type for modifying clientv3.Config
type CfgOption func(*clientv3.Config)
type addrConn struct {
	conn        *grpc.ClientConn
	addr        string
	isConnected bool
}

// SvcDiscoveryRegistryImpl implementation
type SvcDiscoveryRegistryImpl struct {
	client            *clientv3.Client
	resolver          gresolver.Builder
	dialOptions       []grpc.DialOption
	serviceKey        string
	endpointMgr       endpoints.Manager
	leaseID           clientv3.LeaseID
	rpcRegisterTarget string
	watchNames        []string

	rootDirectory string

	mu                 sync.RWMutex
	connMap            map[string][]*addrConn
	serviceDialOptions map[string][]grpc.DialOption
	serviceWatchMu     sync.Mutex
	serviceWatchers    map[string]context.CancelFunc
	watchKeyMu         sync.Mutex
	watchKeyEntries    map[string]*watchKeyEntry

	regMu             sync.Mutex
	keepAliveCancel   context.CancelFunc
	kvKeepAliveMu     sync.Mutex
	kvKeepAliveCancel []context.CancelFunc
	registeredService string
	registeredHost    string
	registeredPort    int
}

func createNoOpLogger() *zap.Logger {
	// Create a no-op write syncer
	noOpWriter := zapcore.AddSync(io.Discard)

	// Create a basic zap core with the no-op writer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		noOpWriter,
		zapcore.InfoLevel, // You can set this to any level that suits your needs
	)

	// Create the logger using the core
	return zap.New(core)
}

// NewSvcDiscoveryRegistry creates a new service discovery registry implementation
func NewSvcDiscoveryRegistry(rootDirectory string, endpoints []string, watchNames []string, options ...CfgOption) (*SvcDiscoveryRegistryImpl, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		// Increase keep-alive queue capacity and message size
		PermitWithoutStream: true,
		Logger:              createNoOpLogger(),
		MaxCallSendMsgSize:  10 * 1024 * 1024, // 10 MB
	}

	// Apply provided options to the config
	for _, opt := range options {
		opt(&cfg)
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	r, err := resolver.NewBuilder(client)
	if err != nil {
		return nil, err
	}

	s := &SvcDiscoveryRegistryImpl{
		client:             client,
		resolver:           r,
		rootDirectory:      rootDirectory,
		connMap:            make(map[string][]*addrConn),
		serviceDialOptions: make(map[string][]grpc.DialOption),
		watchNames:         watchNames,
		serviceWatchers:    make(map[string]context.CancelFunc),
		watchKeyEntries:    make(map[string]*watchKeyEntry),
	}

	s.watchServiceChanges()
	return s, nil
}

func (r *SvcDiscoveryRegistryImpl) GetClient() *clientv3.Client {
	return r.client
}
