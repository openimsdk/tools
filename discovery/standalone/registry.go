package standalone

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
)

type serverHandler func(ctx context.Context, req any, interceptor grpc.UnaryServerInterceptor) (any, error)

func newRegistry() *registry {
	return &registry{
		methods:    make(map[string]serverHandler),
		serializer: newProtoSerializer(),
	}
}

type registry struct {
	lock       sync.RWMutex
	methods    map[string]serverHandler
	serializer serializer
}

func (x *registry) RegisterService(desc *grpc.ServiceDesc, impl any) {
	x.lock.Lock()
	defer x.lock.Unlock()
	for i := range desc.Methods {
		method := desc.Methods[i]
		name := fmt.Sprintf("/%s/%s", desc.ServiceName, method.MethodName)
		if _, ok := x.methods[name]; ok {
			panic(fmt.Errorf("service %s already registered, method %s", desc.ServiceName, method.MethodName))
		}
		x.methods[name] = func(ctx context.Context, req any, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return method.Handler(impl, ctx, func(in any) error {
				tmp, err := x.serializer.Marshal(req)
				if err != nil {
					return err
				}
				return x.serializer.Unmarshal(tmp, in)
			}, interceptor)
		}
	}
}

func (x *registry) getMethod(name string) serverHandler {
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.methods[name]
}
