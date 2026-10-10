package inprocess

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	"github.com/openimsdk/tools/discovery"
)

var (
	global = newServiceRegistry()

	_ discovery.SvcDiscoveryRegistry = (*serviceRegistry)(nil)
	_ grpc.ServiceRegistrar          = (*serviceRegistry)(nil)
)

func newServiceRegistry() *serviceRegistry {
	manager := newConnManager()
	return &serviceRegistry{
		connManager:      manager,
		ServiceRegistrar: manager.local.methods,
	}
}

func GetDiscoveryConn() discovery.Conn {
	return global
}

func GetServiceRegistrar() grpc.ServiceRegistrar {
	return global
}

func GetKeyValue() discovery.KeyValue {
	return global
}

func GetSvcDiscoveryRegistry() discovery.SvcDiscoveryRegistry {
	return global
}

type serviceRegistry struct {
	*connManager
	grpc.ServiceRegistrar
	memoryKV
}

func (r *serviceRegistry) AddOption(...grpc.DialOption) {
	// In-process calls do not dial, so dial options are accepted only for
	// interface compatibility with network discovery implementations.
}

func (r *serviceRegistry) Register(ctx context.Context, serviceName, host string, port int, _ ...grpc.DialOption) error {
	serviceName = strings.TrimSpace(serviceName)
	host = strings.TrimSpace(host)
	if serviceName == "" {
		return fmt.Errorf("inprocess register service name is empty")
	}
	if host == "" {
		return fmt.Errorf("inprocess register host is empty")
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("inprocess register invalid port %d", port)
	}
	target := net.JoinHostPort(host, strconv.Itoa(port))
	return r.registerLocalTarget(ctx, serviceName, target)
}

func (r *serviceRegistry) Close() {
	r.connManager.local.methods.close()
	r.connManager.close()
	r.memoryKV.close()
}

// SetBroadcastAddress enables cross-instance calls: GetConns returns the local
// in-process connection plus one remote connection per address reported by fn.
// secret must equal the peers' share.secret, which their RpcInvoke endpoint checks.
func SetBroadcastAddress(secret string, fn BroadcastAddressProvider) {
	global.setBroadcastAddressProvider(secret, fn)
}

func SetLocalTarget(addr string) {
	global.setLocalTarget(addr)
}

func SetContextAdminFunc(fn func(ctx context.Context) bool) {
	contextAdminFunc.Store(&fn)
}
