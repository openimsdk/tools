package etcd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"

	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
)

type connPool struct {
	client          *clientv3.Client
	rootDirectory   string
	resolverBuilder resolver.Builder

	mu                 sync.RWMutex
	endpointConnMap    map[string][]*grpc.ClientConn // fullServiceKey -> []*ClientConn
	serviceConns       map[string]*grpc.ClientConn
	dialOptions        []grpc.DialOption
	serviceDialOptions map[string][]grpc.DialOption
	connectionsClosed  bool

	watchersMu sync.Mutex
	watchers   map[string]*serviceWatcher
	closed     bool
}

func newConnPool(client *clientv3.Client, rootDirectory string, watchNames []string) *connPool {
	cp := &connPool{
		client:             client,
		rootDirectory:      rootDirectory,
		endpointConnMap:    make(map[string][]*grpc.ClientConn),
		serviceConns:       make(map[string]*grpc.ClientConn),
		serviceDialOptions: make(map[string][]grpc.DialOption),
		watchers:           make(map[string]*serviceWatcher),
	}
	cp.resolverBuilder = serviceResolverBuilder{pool: cp}
	for _, service := range watchNames {
		if _, err := cp.getOrCreateWatcher(cp.combineKeyWithPrefix(service)); err != nil {
			log.ZWarn(context.Background(), "ensure service watcher err", err, zap.String("service", service))
		}
	}
	return cp
}

func (cp *connPool) combineKeyWithPrefix(key string) string {
	return fmt.Sprintf("%s/%s", cp.rootDirectory, key)
}

func (cp *connPool) getOrCreateWatcher(prefix string) (*serviceWatcher, error) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	cp.watchersMu.Lock()
	defer cp.watchersMu.Unlock()

	if cp.closed || cp.client == nil {
		return nil, fmt.Errorf("etcd client closed")
	}
	if watch, ok := cp.watchers[prefix]; ok {
		return watch, nil
	}

	serviceName := strings.TrimSuffix(strings.TrimPrefix(prefix, cp.rootDirectory+"/"), "/")
	watcher := newServiceWatcher(cp.client, prefix)
	watcher.addListener(func(addresses []resolver.Address, err error) {
		if err == nil {
			cp.syncEndpointConnMap(serviceName, addresses)
		}
	})
	cp.watchers[prefix] = watcher
	return watcher, nil
}

func (cp *connPool) syncEndpointConnMap(serviceName string, addresses []resolver.Address) {
	fullServiceKey := cp.combineKeyWithPrefix(serviceName)
	cp.mu.Lock()
	if cp.connectionsClosed {
		cp.mu.Unlock()
		return
	}
	old := make(map[string]*grpc.ClientConn)
	for _, conn := range cp.endpointConnMap[fullServiceKey] {
		old[conn.Target()] = conn
	}
	var current []*grpc.ClientConn
	var toClose []*grpc.ClientConn
	dialOpts := datautil.Flatten(cp.dialOptions, cp.serviceDialOptions[fullServiceKey], []grpc.DialOption{grpc.WithResolvers(cp.resolverBuilder)})
	seen := make(map[string]struct{}, len(addresses))
	for _, address := range addresses {
		addr := address.Addr
		if addr == "" {
			continue
		}
		if _, duplicate := seen[addr]; duplicate {
			continue
		}
		seen[addr] = struct{}{}
		if conn, ok := old[addr]; ok {
			current = append(current, conn)
			delete(old, addr)
			continue
		}
		if conn, err := grpc.NewClient(addr, dialOpts...); err != nil {
			log.ZWarn(context.Background(), "failed to dial new endpoint", err, zap.String("service", serviceName), zap.String("addr", addr))
		} else {
			current = append(current, conn)
		}
	}
	for _, conn := range old {
		toClose = append(toClose, conn)
	}
	cp.endpointConnMap[fullServiceKey] = current
	cp.mu.Unlock()
	for _, conn := range toClose {
		if err := conn.Close(); err != nil {
			log.ZWarn(context.Background(), "failed to close stale conn", err, zap.String("service", serviceName), zap.String("addr", conn.Target()))
		}
	}
}

// GetConns returns gRPC client connections for a given service name
func (cp *connPool) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error) {
	fullServiceKey := cp.combineKeyWithPrefix(serviceName)
	if len(opts) > 0 {
		cp.mu.Lock()
		cp.serviceDialOptions[fullServiceKey] = datautil.CopySlice(opts)
		cp.mu.Unlock()
	}
	watcher, err := cp.getOrCreateWatcher(cp.combineKeyWithPrefix(serviceName))
	if err != nil {
		return nil, err
	}

	waitCtx, cancel := withTimeout(ctx, 3*time.Second)
	defer cancel()
	_ = watcher.waitReady(waitCtx)

	cp.mu.RLock()
	conns := cp.endpointConnMap[fullServiceKey]
	res := datautil.Slice(conns, func(e *grpc.ClientConn) grpc.ClientConnInterface { return e })
	cp.mu.RUnlock()

	return res, nil
}

// GetConn returns a single gRPC client connection for a given service name
func (cp *connPool) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error) {
	if _, err := cp.getOrCreateWatcher(cp.combineKeyWithPrefix(serviceName)); err != nil {
		return nil, err
	}
	target := fmt.Sprintf("etcd:///%s", cp.combineKeyWithPrefix(serviceName))
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.connectionsClosed {
		return nil, fmt.Errorf("etcd connection pool closed")
	}
	if conn := cp.serviceConns[target]; conn != nil {
		return conn, nil
	}
	dialOpts := datautil.Flatten(cp.dialOptions, opts, []grpc.DialOption{grpc.WithResolvers(cp.resolverBuilder)})
	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, err
	}
	cp.serviceConns[target] = conn
	return conn, nil
}

// AddOption appends gRPC dial options to the existing options
func (cp *connPool) AddOption(opts ...grpc.DialOption) {
	cp.mu.Lock()
	toClose := cp.resetConnMapLocked()
	cp.dialOptions = append(cp.dialOptions, opts...)
	cp.mu.Unlock()

	ctx := context.Background()
	for _, c := range toClose {
		if err := c.Close(); err != nil {
			log.ZWarn(ctx, "failed to close conn", err)
		}
	}

	cp.watchersMu.Lock()
	type syncItem struct {
		serviceName string
		addrs       []resolver.Address
	}
	var toSync []syncItem
	for prefix, w := range cp.watchers {
		serviceName := strings.TrimPrefix(prefix, cp.rootDirectory+"/")
		serviceName = strings.TrimSuffix(serviceName, "/")
		if addrs := w.getAddresses(); len(addrs) > 0 {
			toSync = append(toSync, syncItem{serviceName: serviceName, addrs: addrs})
		}
	}
	cp.watchersMu.Unlock()

	for _, item := range toSync {
		cp.syncEndpointConnMap(item.serviceName, item.addrs)
	}
}

func (cp *connPool) resetConnMapLocked() []*grpc.ClientConn {
	var toClose []*grpc.ClientConn
	for _, conns := range cp.endpointConnMap {
		toClose = append(toClose, conns...)
	}
	cp.endpointConnMap = make(map[string][]*grpc.ClientConn)
	return toClose
}

func (cp *connPool) close() {
	cp.watchersMu.Lock()
	cp.closed = true
	watchers := datautil.Values(cp.watchers)
	cp.watchers = nil
	cp.watchersMu.Unlock()
	for _, w := range watchers {
		w.close()
	}

	cp.mu.Lock()
	cp.connectionsClosed = true
	toClose := cp.resetConnMapLocked()
	for _, conn := range cp.serviceConns {
		toClose = append(toClose, conn)
	}
	cp.serviceConns = nil
	cp.mu.Unlock()
	for _, c := range toClose {
		_ = c.Close()
	}
}
