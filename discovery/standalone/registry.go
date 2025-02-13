package standalone

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	lock       sync.Mutex
	methods    map[string]serverHandler
	serializer serializer
	wait       map[string]chan struct{}
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
		if wait, ok := x.wait[name]; ok {
			delete(x.wait, name)
			close(wait)
		}
	}
}

func (x *registry) getMethod(ctx context.Context, name string) (serverHandler, error) {
	x.lock.Lock()
	handler, ok := x.methods[name]
	if ok {
		x.lock.Unlock()
		return handler, nil
	}
	if x.wait == nil {
		x.wait = make(map[string]chan struct{})
	}
	wait, ok := x.wait[name]
	if !ok {
		wait = make(chan struct{})
		x.wait[name] = wait
	}
	x.lock.Unlock()
	timeout := time.NewTimer(time.Second * 30)
	defer timeout.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timeout.C:
		return nil, fmt.Errorf("get service %s timeout", name)
	case <-wait:
		x.lock.Lock()
		handler, ok = x.methods[name]
		x.lock.Unlock()
		if !ok {
			return nil, fmt.Errorf("get service %s internal error", name)
		}
		return handler, nil
	}
}
