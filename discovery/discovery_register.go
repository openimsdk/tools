package discovery

import (
	"context"
	"errors"
	"strconv"

	"google.golang.org/grpc"
)

var ErrNotSupported = errors.New("discovery data not supported")

type Conn interface {
	GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (grpc.ClientConnInterface, error)
	GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]grpc.ClientConnInterface, error)
	IsSelfNode(cc grpc.ClientConnInterface) bool
}

type WatchType int

func (t WatchType) String() string {
	switch t {
	case WatchTypePut:
		return "PUT"
	case WatchTypeDelete:
		return "DELETE"
	default:
		return strconv.Itoa(int(t))
	}
}

const (
	WatchTypePut    WatchType = 0
	WatchTypeDelete WatchType = 1
)

type WatchKey struct {
	Key   []byte
	Value []byte
	Type  WatchType
}

type WatchKeyHandler func(data *WatchKey) error

type KeyValue interface {
	SetKey(ctx context.Context, key string, value []byte) error
	SetWithLease(ctx context.Context, key string, val []byte, ttl int64) error
	GetKey(ctx context.Context, key string) ([]byte, error)
	GetKeyWithPrefix(ctx context.Context, key string) ([][]byte, error)
	WatchKey(ctx context.Context, key string, fn WatchKeyHandler) error
}

type SvcDiscoveryRegistry interface {
	Conn
	KeyValue
	AddOption(opts ...grpc.DialOption)
	Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error
	Close()
	GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error)
}
