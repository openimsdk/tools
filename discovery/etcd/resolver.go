package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc/resolver"
)

type resolverBuilder struct {
	client *clientv3.Client
}

func (builder resolverBuilder) Scheme() string { return "etcd" }

func (builder resolverBuilder) Build(target resolver.Target, conn resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	if target.Endpoint() == "" {
		return nil, errors.New("etcd resolver: empty target")
	}
	ctx, cancel := context.WithCancel(builder.client.Ctx())
	watcher := &endpointResolver{
		client: builder.client,
		prefix: target.Endpoint() + "/",
		conn:   conn,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go watcher.run(ctx)
	return watcher, nil
}

type endpointResolver struct {
	client *clientv3.Client
	prefix string
	conn   resolver.ClientConn
	cancel context.CancelFunc
	done   chan struct{}
}

func (watcher *endpointResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (watcher *endpointResolver) Close() {
	watcher.cancel()
	<-watcher.done
}

func (watcher *endpointResolver) run(ctx context.Context) {
	defer close(watcher.done)
	delay := 100 * time.Millisecond
	for ctx.Err() == nil {
		err := watcher.watchSnapshot(ctx, func() { delay = 100 * time.Millisecond })
		if ctx.Err() != nil {
			return
		}
		watcher.conn.ReportError(err)
		if !sleepWithContext(ctx, delay) {
			return
		}
		delay = min(delay*2, 3*time.Second)
	}
}

func (watcher *endpointResolver) watchSnapshot(parent context.Context, recovered func()) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	getCtx, getCancel := context.WithTimeout(ctx, 5*time.Second)
	snapshot, err := watcher.client.Get(getCtx, watcher.prefix, clientv3.WithPrefix())
	getCancel()
	if err != nil {
		return err
	}
	addresses := make(map[string]resolver.Address, len(snapshot.Kvs))
	for _, entry := range snapshot.Kvs {
		watcher.setAddress(addresses, string(entry.Key), entry.Value)
	}
	watcher.publish(addresses)
	updates := watcher.client.Watch(ctx, watcher.prefix, clientv3.WithPrefix(), clientv3.WithRev(snapshot.Header.Revision+1), clientv3.WithProgressNotify())
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, open := <-updates:
			if !open {
				return errors.New("etcd resolver: watch closed")
			}
			if err := update.Err(); err != nil {
				return err
			}
			if len(update.Events) > 0 || update.IsProgressNotify() {
				recovered()
			}
			for _, event := range update.Events {
				switch event.Type {
				case clientv3.EventTypePut:
					watcher.setAddress(addresses, string(event.Kv.Key), event.Kv.Value)
				case clientv3.EventTypeDelete:
					delete(addresses, string(event.Kv.Key))
				}
			}
			if len(update.Events) > 0 {
				watcher.publish(addresses)
			}
		}
	}
}

func (watcher *endpointResolver) setAddress(addresses map[string]resolver.Address, key string, value []byte) {
	var endpoint endpoints.Endpoint
	if err := json.Unmarshal(value, &endpoint); err != nil {
		watcher.conn.ReportError(err)
		return
	}
	addresses[key] = resolver.Address{Addr: endpoint.Addr, Metadata: endpoint.Metadata}
}

func (watcher *endpointResolver) publish(addresses map[string]resolver.Address) {
	keys := make([]string, 0, len(addresses))
	for key := range addresses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	state := resolver.State{Addresses: make([]resolver.Address, 0, len(keys))}
	for _, key := range keys {
		state.Addresses = append(state.Addresses, addresses[key])
	}
	if err := watcher.conn.UpdateState(state); err != nil {
		watcher.conn.ReportError(err)
	}
}
