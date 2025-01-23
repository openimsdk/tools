package standalone

import (
	"github.com/openimsdk/tools/discovery"
	"google.golang.org/grpc"
)

var globalDiscoveryConn = newDiscoveryConn()

func GetDiscoveryConn() discovery.Conn {
	return globalDiscoveryConn
}

func GetServiceRegistrar() grpc.ServiceRegistrar {
	return globalDiscoveryConn.conn.registry
}
