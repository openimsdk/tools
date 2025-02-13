package standalone

import (
	"context"

	"github.com/openimsdk/tools/discovery"
	"google.golang.org/grpc"
)

var global *svcDiscoveryRegistry

func init() {
	conn := newDiscoveryConn()
	global = &svcDiscoveryRegistry{
		Conn:             conn,
		ServiceRegistrar: conn.conn.registry,
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

type svcDiscoveryRegistry struct {
	discovery.Conn
	grpc.ServiceRegistrar
	keyValue
}

func (x *svcDiscoveryRegistry) AddOption(opts ...grpc.DialOption) {}

func (x *svcDiscoveryRegistry) Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error {
	return nil
}

func (x *svcDiscoveryRegistry) Close() {}

func (x *svcDiscoveryRegistry) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	return "", nil
}
