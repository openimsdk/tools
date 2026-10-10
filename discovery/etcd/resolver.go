package etcd

import (
	"errors"
	"strings"

	"google.golang.org/grpc/resolver"
)

type serviceResolverBuilder struct {
	pool *connPool
}

func (builder serviceResolverBuilder) Scheme() string { return "etcd" }

func (builder serviceResolverBuilder) Build(target resolver.Target, conn resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	if target.Endpoint() == "" {
		return nil, errors.New("etcd resolver: empty target")
	}
	prefix := target.Endpoint()
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	watch, err := builder.pool.getOrCreateWatcher(prefix)
	if err != nil {
		return nil, err
	}
	r := &serviceResolver{watch: watch, conn: conn}
	r.id = watch.addListener(r.consume)
	return r, nil
}

type serviceResolver struct {
	watch *serviceWatcher
	conn  resolver.ClientConn
	id    uint64
}

func (r *serviceResolver) consume(addresses []resolver.Address, err error) {
	if err != nil {
		r.conn.ReportError(err)
		return
	}
	if err := r.conn.UpdateState(resolver.State{Addresses: addresses}); err != nil {
		r.conn.ReportError(err)
	}
}

func (*serviceResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *serviceResolver) Close() { r.watch.removeListener(r.id) }
