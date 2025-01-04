package etcd

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	gresolver "google.golang.org/grpc/resolver"
)

// ZkOption defines a function type for modifying clientv3.Config
type ZkOption func(*clientv3.Config)
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

	mu      sync.RWMutex
	connMap map[string][]*addrConn
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
func NewSvcDiscoveryRegistry(rootDirectory string, endpoints []string, watchNames []string, options ...ZkOption) (*SvcDiscoveryRegistryImpl, error) {
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
		client:        client,
		resolver:      r,
		rootDirectory: rootDirectory,
		connMap:       make(map[string][]*addrConn),
		watchNames:    watchNames,
	}

	s.watchServiceChanges()
	return s, nil
}

// initializeConnMap fetches all existing endpoints and populates the local map
func (r *SvcDiscoveryRegistryImpl) initializeConnMap(opts ...grpc.DialOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx := context.Background()
	for _, name := range r.watchNames {
		fullPrefix := fmt.Sprintf("%s/%s", r.rootDirectory, name)
		resp, err := r.client.Get(ctx, fullPrefix, clientv3.WithPrefix())
		if err != nil {
			return err
		}

		oldList := r.connMap[fullPrefix]

		addrMap := make(map[string]*addrConn, len(oldList))
		for _, conn := range oldList {
			addrMap[conn.addr] = conn
		}
		newList := make([]*addrConn, 0, len(oldList))
		for _, kv := range resp.Kvs {
			prefix, addr := r.splitEndpoint(string(kv.Key))
			if addr == "" {
				continue
			}
			if _, _, err = net.SplitHostPort(addr); err != nil {
				continue
			}
			if prefix != fullPrefix {
				continue
			}

			if conn, ok := addrMap[addr]; ok {
				conn.isConnected = true
				continue
			}

			dialOpts := append(append(r.dialOptions, opts...), grpc.WithResolvers(r.resolver))

			err := r.checkOpts(dialOpts...) // Check opts in include mw.GrpcClient()
			if err != nil {
				return errs.WrapMsg(err, "checkOpts is failed")
			}

			conn, err := grpc.DialContext(context.Background(), addr, dialOpts...)
			if err != nil {
				continue
			}
			newList = append(newList, &addrConn{conn: conn, addr: addr, isConnected: false})
		}
		for _, conn := range oldList {
			if conn.isConnected {
				conn.isConnected = false
				newList = append(newList, conn)
				continue
			}
			if err = conn.conn.Close(); err != nil {
				log.ZWarn(ctx, "close conn err", err)
			}
		}
		r.connMap[fullPrefix] = newList
	}

	return nil
}

// WithDialTimeout sets a custom dial timeout for the etcd client
func WithDialTimeout(timeout time.Duration) ZkOption {
	return func(cfg *clientv3.Config) {
		cfg.DialTimeout = timeout
	}
}

// WithMaxCallSendMsgSize sets a custom max call send message size for the etcd client
func WithMaxCallSendMsgSize(size int) ZkOption {
	return func(cfg *clientv3.Config) {
		cfg.MaxCallSendMsgSize = size
	}
}

// WithUsernameAndPassword sets a username and password for the etcd client
func WithUsernameAndPassword(username, password string) ZkOption {
	return func(cfg *clientv3.Config) {
		cfg.Username = username
		cfg.Password = password
	}
}

// GetUserIdHashGatewayHost returns the gateway host for a given user ID hash
func (r *SvcDiscoveryRegistryImpl) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	return "", nil
}

// GetConns returns gRPC client connections for a given service name
func (r *SvcDiscoveryRegistryImpl) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]*grpc.ClientConn, error) {
	fullServiceKey := fmt.Sprintf("%s/%s", r.rootDirectory, serviceName)
	if len(r.connMap) == 0 {
		if err := r.initializeConnMap(opts...); err != nil {
			return nil, err
		}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return datautil.Batch(func(t *addrConn) *grpc.ClientConn { return t.conn }, r.connMap[fullServiceKey]), nil
}

// GetConn returns a single gRPC client connection for a given service name
func (r *SvcDiscoveryRegistryImpl) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	target := fmt.Sprintf("etcd:///%s/%s", r.rootDirectory, serviceName)

	dialOpts := append(append(r.dialOptions, opts...), grpc.WithResolvers(r.resolver))

	err := r.checkOpts(dialOpts...) // Check opts in include mw.GrpcClient()
	if err != nil {
		return nil, errs.WrapMsg(err, "checkOpts is failed")
	}

	return grpc.DialContext(ctx, target, dialOpts...)
}

// GetSelfConnTarget returns the connection target for the current service
func (r *SvcDiscoveryRegistryImpl) GetSelfConnTarget() string {
	return r.rpcRegisterTarget
}

// AddOption appends gRPC dial options to the existing options
func (r *SvcDiscoveryRegistryImpl) AddOption(opts ...grpc.DialOption) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetConnMap()
	r.dialOptions = append(r.dialOptions, opts...)
}

// CloseConn closes a given gRPC client connection
func (r *SvcDiscoveryRegistryImpl) CloseConn(conn *grpc.ClientConn) {
	conn.Close()
}

// Register registers a new service endpoint with etcd
func (r *SvcDiscoveryRegistryImpl) Register(serviceName, host string, port int, opts ...grpc.DialOption) error {
	r.serviceKey = fmt.Sprintf("%s/%s/%s:%d", r.rootDirectory, serviceName, host, port)
	em, err := endpoints.NewManager(r.client, r.rootDirectory+"/"+serviceName)
	if err != nil {
		return err
	}
	r.endpointMgr = em

	leaseResp, err := r.client.Grant(context.Background(), 30) //
	if err != nil {
		return err
	}
	r.leaseID = leaseResp.ID

	r.rpcRegisterTarget = fmt.Sprintf("%s:%d", host, port)
	endpoint := endpoints.Endpoint{Addr: r.rpcRegisterTarget}

	err = em.AddEndpoint(context.TODO(), r.serviceKey, endpoint, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return err
	}

	go r.keepAliveLease(r.leaseID)
	return nil
}

// keepAliveLease maintains the lease alive by sending keep-alive requests
func (r *SvcDiscoveryRegistryImpl) keepAliveLease(leaseID clientv3.LeaseID) {
	ch, err := r.client.KeepAlive(context.Background(), leaseID)
	if err != nil {
		return
	}
	for ka := range ch {
		if ka != nil {
		} else {
			return
		}
	}
}

// watchServiceChanges watches for changes in the service directory
func (r *SvcDiscoveryRegistryImpl) watchServiceChanges() {
	for _, s := range r.watchNames {
		go func() {
			watchChan := r.client.Watch(context.Background(), r.rootDirectory+"/"+s, clientv3.WithPrefix())
			for range watchChan {
				if err := r.initializeConnMap(); err != nil {
					log.ZWarn(context.Background(), "initializeConnMap in watch err", err)
				}
			}
		}()
	}
}

// splitEndpoint splits the endpoint string into prefix and address
func (r *SvcDiscoveryRegistryImpl) splitEndpoint(input string) (string, string) {
	lastSlashIndex := strings.LastIndex(input, "/")
	if lastSlashIndex != -1 {
		part1 := input[:lastSlashIndex]
		part2 := input[lastSlashIndex+1:]
		return part1, part2
	}
	return input, ""
}

// UnRegister removes the service endpoint from etcd
func (r *SvcDiscoveryRegistryImpl) UnRegister() error {
	if r.endpointMgr == nil {
		return fmt.Errorf("endpoint manager is not initialized")
	}
	err := r.endpointMgr.DeleteEndpoint(context.TODO(), r.serviceKey)
	if err != nil {
		return err
	}
	return nil
}

// Close closes the etcd client connection
func (r *SvcDiscoveryRegistryImpl) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetConnMap()
	if r.client != nil {
		_ = r.client.Close()
	}
}

// Check verifies if etcd is running by checking the existence of the root node and optionally creates it with a lease
func Check(ctx context.Context, etcdServers []string, etcdRoot string, createIfNotExist bool, options ...ZkOption) error {
	cfg := clientv3.Config{
		Endpoints: etcdServers,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	client, err := clientv3.New(cfg)
	if err != nil {
		return errs.WrapMsg(err, "failed to connect to etcd")
	}
	defer client.Close()

	var opCtx context.Context
	var cancel context.CancelFunc
	if cfg.DialTimeout != 0 {
		opCtx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
	} else {
		opCtx, cancel = context.WithTimeout(ctx, 10*time.Second)
	}
	defer cancel()

	resp, err := client.Get(opCtx, etcdRoot)
	if err != nil {
		return errs.WrapMsg(err, "failed to get the root node from etcd")
	}

	if len(resp.Kvs) == 0 {
		if createIfNotExist {
			var leaseTTL int64 = 10
			var leaseResp *clientv3.LeaseGrantResponse
			if leaseTTL > 0 {
				leaseResp, err = client.Grant(opCtx, leaseTTL)
				if err != nil {
					return errs.WrapMsg(err, "failed to create lease in etcd")
				}
			}
			putOpts := []clientv3.OpOption{}
			if leaseResp != nil {
				putOpts = append(putOpts, clientv3.WithLease(leaseResp.ID))
			}

			_, err := client.Put(opCtx, etcdRoot, "", putOpts...)
			if err != nil {
				return errs.WrapMsg(err, "failed to create the root node in etcd")
			}
		} else {
			return fmt.Errorf("root node %s does not exist in etcd", etcdRoot)
		}
	}
	return nil
}

func (r *SvcDiscoveryRegistryImpl) GetClient() *clientv3.Client {
	return r.client
}

func (r *SvcDiscoveryRegistryImpl) checkOpts(opts ...grpc.DialOption) error {
	// mwOpt := mw.GrpcClient()

	// for _, opt := range opts {
	// 	if opt == mwOpt {
	// 		return nil
	// 	}
	// }

	// return errs.New("missing required grpc.DialOption", "option", "mw.GrpcClient")
	return nil
}

func (r *SvcDiscoveryRegistryImpl) resetConnMap() {
	ctx := context.Background()
	for _, conn := range r.connMap {
		for _, c := range conn {
			if err := c.conn.Close(); err != nil {
				log.ZWarn(ctx, "failed to close conn", err)
			}
		}
	}
	r.connMap = make(map[string][]*addrConn)
}
