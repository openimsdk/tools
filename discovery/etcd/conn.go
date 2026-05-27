package etcd

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// initializeConnMap fetches all existing endpoints for the given service and populates the local map
func (r *SvcDiscoveryRegistryImpl) initializeConnMap(service string, opts ...grpc.DialOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.client == nil {
		return fmt.Errorf("etcd client closed")
	}

	ctx := context.Background()
	fullPrefix := r.combineKeyWithPrefix(service)
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

		dialOpts := append([]grpc.DialOption{}, r.dialOptions...)
		if storedOpts, ok := r.serviceDialOptions[fullPrefix]; ok && len(storedOpts) > 0 {
			dialOpts = append(dialOpts, storedOpts...)
		} else if len(opts) > 0 {
			dialOpts = append(dialOpts, opts...)
		}
		dialOpts = append(dialOpts, grpc.WithResolvers(r.resolver))

		err := r.checkOpts(dialOpts...) // Check opts in include mw.GrpcClient()
		if err != nil {
			return errs.WrapMsg(err, "checkOpts is failed")
		}

		conn, err := grpc.NewClient(addr, dialOpts...)
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

	return nil
}

// GetUserIdHashGatewayHost returns the gateway host for a given user ID hash
func (r *SvcDiscoveryRegistryImpl) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	return "", nil
}

// GetConns returns gRPC client connections for a given service name
func (r *SvcDiscoveryRegistryImpl) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error) {
	if err := r.ensureServiceWatch(serviceName); err != nil {
		return nil, err
	}

	fullServiceKey := r.combineKeyWithPrefix(serviceName)

	if len(opts) > 0 {
		r.mu.Lock()
		r.serviceDialOptions[fullServiceKey] = append([]grpc.DialOption(nil), opts...)
		r.mu.Unlock()
	}

	r.mu.RLock()
	existing := r.connMap[fullServiceKey]
	r.mu.RUnlock()

	if len(existing) == 0 {
		if err := r.initializeConnMap(serviceName, opts...); err != nil {
			return nil, err
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return datautil.Batch(func(t *addrConn) grpc.ClientConnInterface { return t.conn }, r.connMap[fullServiceKey]), nil
}

// GetConn returns a single gRPC client connection for a given service name
func (r *SvcDiscoveryRegistryImpl) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error) {
	target := fmt.Sprintf("etcd:///%s", r.combineKeyWithPrefix(serviceName))

	dialOpts := append(append(r.dialOptions, opts...), grpc.WithResolvers(r.resolver))

	err := r.checkOpts(dialOpts...) // Check opts in include mw.GrpcClient()
	if err != nil {
		return nil, errs.WrapMsg(err, "checkOpts is failed")
	}

	return grpc.NewClient(target, dialOpts...)
}

// GetSelfConnTarget returns the connection target for the current service
func (r *SvcDiscoveryRegistryImpl) GetSelfConnTarget() string {
	return r.rpcRegisterTarget
}

func (r *SvcDiscoveryRegistryImpl) IsSelfNode(cc grpc.ClientConnInterface) bool {
	cli, ok := cc.(*grpc.ClientConn)
	if !ok {
		return false
	}
	return r.GetSelfConnTarget() == cli.Target()
}

// AddOption appends gRPC dial options to the existing options
func (r *SvcDiscoveryRegistryImpl) AddOption(opts ...grpc.DialOption) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetConnMap()
	r.dialOptions = append(r.dialOptions, opts...)
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
