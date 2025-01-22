package standalone

import "google.golang.org/grpc"

var globalClientConn = newClientConn()

func GetClientConn() grpc.ClientConnInterface {
	return globalClientConn
}

func GetServiceRegistrar() grpc.ServiceRegistrar {
	return globalClientConn.Registry()
}
