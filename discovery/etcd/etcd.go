package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
)

const (
	defaultRegisterTimeout = 5 * time.Second
	defaultLeaseTTL        = int64(30)
	keepAliveRetryDelay    = time.Second
	defaultCloseTimeout    = 5 * time.Second
)

// SvcDiscoveryRegistryImpl implementation
type SvcDiscoveryRegistryImpl struct {
	client        *clientv3.Client
	rootDirectory string

	*registrar
	*connPool
	*kvHub
}

// NewSvcDiscoveryRegistry creates a new service discovery registry implementation
func NewSvcDiscoveryRegistry(rootDirectory string, endpoints []string, watchNames []string, options ...CfgOption) (*SvcDiscoveryRegistryImpl, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		// Increase keep-alive queue capacity and message size
		PermitWithoutStream: true,
		Logger:              zap.NewNop(),
		MaxCallSendMsgSize:  10 * 1024 * 1024, // 10 MB
	}

	// Apply provided options to the config
	datautil.Foreach(options, func(option CfgOption) { option(&cfg) })

	cfg.DialOptions = append(cfg.DialOptions, grpc.WithStatsHandler(etcdRPCDiagnostics{}))

	client, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	client.Watcher = newReconnectingWatcher(client, client.Watcher)

	log.ZInfo(context.Background(), "etcd client created",
		"endpoints", cfg.Endpoints,
		"rootDirectory", rootDirectory,
		"authConfigured", cfg.Username != "",
	)

	return &SvcDiscoveryRegistryImpl{
		client:        client,
		rootDirectory: rootDirectory,
		registrar:     newRegistrar(client, rootDirectory),
		connPool:      newConnPool(client, rootDirectory, watchNames),
		kvHub:         newKVHub(client, rootDirectory),
	}, nil
}

func (r *SvcDiscoveryRegistryImpl) GetClient() *clientv3.Client {
	return r.client
}

func (r *SvcDiscoveryRegistryImpl) Close() {
	if r.connPool != nil {
		r.connPool.close()
	}
	if r.kvHub != nil {
		r.kvHub.close()
	}
	if r.registrar != nil {
		_ = r.registrar.UnRegister()
	}
	if r.client != nil {
		_ = r.client.Close()
	}
}
