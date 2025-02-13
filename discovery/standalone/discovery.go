package standalone

import (
	"context"

	"google.golang.org/grpc"
)

func newDiscoveryConn() *discoveryConn {
	return &discoveryConn{
		conn: newClientConn(),
	}
}

type discoveryConn struct {
	conn *clientConn
}

func (x discoveryConn) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error) {
	return x.conn, nil
}

func (x discoveryConn) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error) {
	return []grpc.ClientConnInterface{x.conn}, nil
}

func (x discoveryConn) IsSelfNode(cc grpc.ClientConnInterface) bool {
	return true
}
