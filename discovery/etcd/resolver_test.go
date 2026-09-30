package etcd

import (
	"context"
	"errors"
	"net/url"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

type resolverKV struct {
	clientv3.KV
	get func(context.Context) (*clientv3.GetResponse, error)
}

func (kv resolverKV) Get(ctx context.Context, _ string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return kv.get(ctx)
}

type resolverWatch struct {
	clientv3.Watcher
	watch func(context.Context, int64) clientv3.WatchChan
}

func (watch resolverWatch) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return watch.watch(ctx, clientv3.OpGet(key, opts...).Rev())
}

type resolverConn struct {
	resolver.ClientConn
	states chan resolver.State
	errors chan error
}

func (conn resolverConn) UpdateState(state resolver.State) error {
	conn.states <- state
	return nil
}

func (conn resolverConn) ReportError(err error) { conn.errors <- err }

func receive[T any](t *testing.T, values <-chan T) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("resolver operation did not complete")
		var zero T
		return zero
	}
}

func Test_Resolver_replaces_snapshot_when_watch_closes(t *testing.T) {
	client := clientv3.NewCtxClient(t.Context())
	var calls atomic.Int64
	client.KV = resolverKV{get: func(context.Context) (*clientv3.GetResponse, error) {
		revision := calls.Add(1)
		response := &clientv3.GetResponse{Header: &etcdserverpb.ResponseHeader{Revision: revision}}
		switch revision {
		case 1:
			response.Kvs = []*mvccpb.KeyValue{{Key: []byte("service/A"), Value: []byte(`{"Op":0,"Addr":"A","Metadata":{"zone":"old"}}`)}}
		case 3:
			response.Kvs = []*mvccpb.KeyValue{{Key: []byte("service/B"), Value: []byte(`{"Op":0,"Addr":"B","Metadata":{"zone":"new"}}`)}}
		}
		return response, nil
	}}
	revisions := make(chan int64, 3)
	active := make(chan context.Context, 1)
	client.Watcher = resolverWatch{watch: func(ctx context.Context, revision int64) clientv3.WatchChan {
		revisions <- revision
		updates := make(chan clientv3.WatchResponse)
		if revision < 4 {
			close(updates)
		} else {
			active <- ctx
		}
		return updates
	}}
	conn := resolverConn{states: make(chan resolver.State, 4), errors: make(chan error, 4)}
	watcher, err := (resolverBuilder{client: client}).Build(resolver.Target{URL: url.URL{Path: "/service"}}, conn, resolver.BuildOptions{})
	require.NoError(t, err)
	t.Cleanup(watcher.Close)

	require.Equal(t, "A", receive(t, conn.states).Addresses[0].Addr)
	require.Empty(t, receive(t, conn.states).Addresses)
	state := receive(t, conn.states)
	require.Len(t, state.Addresses, 1)
	require.Equal(t, "B", state.Addresses[0].Addr)
	require.Equal(t, map[string]interface{}{"zone": "new"}, state.Addresses[0].Metadata)
	for _, revision := range []int64{2, 3, 4} {
		require.Equal(t, revision, receive(t, revisions))
	}
	watchCtx := receive(t, active)
	watcher.Close()
	require.ErrorIs(t, watchCtx.Err(), context.Canceled)
}

func Test_Resolver_cancels_work_when_closed(t *testing.T) {
	for _, phase := range []string{"snapshot", "watch", "backoff", "client"} {
		t.Run(phase, func(t *testing.T) {
			parent, cancel := context.WithCancel(t.Context())
			defer cancel()
			client := clientv3.NewCtxClient(parent)
			entered := make(chan context.Context, 1)
			client.KV = resolverKV{get: func(ctx context.Context) (*clientv3.GetResponse, error) {
				if phase == "snapshot" || phase == "client" {
					entered <- ctx
					<-ctx.Done()
					return nil, ctx.Err()
				}
				if phase == "backoff" {
					return nil, errors.New("snapshot unavailable")
				}
				return &clientv3.GetResponse{Header: &etcdserverpb.ResponseHeader{Revision: 1}}, nil
			}}
			client.Watcher = resolverWatch{watch: func(ctx context.Context, _ int64) clientv3.WatchChan {
				entered <- ctx
				return make(chan clientv3.WatchResponse)
			}}
			conn := resolverConn{states: make(chan resolver.State, 1), errors: make(chan error, 1)}
			watcher, err := (resolverBuilder{client: client}).Build(resolver.Target{URL: url.URL{Path: "/service"}}, conn, resolver.BuildOptions{})
			require.NoError(t, err)
			t.Cleanup(watcher.Close)
			if phase == "backoff" {
				require.EqualError(t, receive(t, conn.errors), "snapshot unavailable")
			} else {
				operation := receive(t, entered)
				t.Cleanup(func() { require.ErrorIs(t, operation.Err(), context.Canceled) })
			}

			closed := make(chan struct{})
			if phase == "client" {
				cancel()
				closed = watcher.(*endpointResolver).done
			} else {
				go func() {
					watcher.Close()
					close(closed)
				}()
			}
			receive(t, closed)
		})
	}
}

func Test_Resolver_backoff_when_watch_recovers(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		updates []clientv3.WatchResponse
		retry   []time.Duration
	}{
		{name: "closed"},
		{name: "created", updates: []clientv3.WatchResponse{{Created: true, Header: etcdserverpb.ResponseHeader{Revision: 8}}}},
		{name: "empty", updates: []clientv3.WatchResponse{{}}},
		{name: "canceled", updates: []clientv3.WatchResponse{{Canceled: true}}},
		{name: "compacted", updates: []clientv3.WatchResponse{{CompactRevision: 8}}},
		{name: "put", updates: []clientv3.WatchResponse{{Events: []*clientv3.Event{{Type: clientv3.EventTypePut, Kv: &mvccpb.KeyValue{Key: []byte("service/A"), Value: []byte(`{"Addr":"A"}`)}}}}}, retry: []time.Duration{100, 200}},
		{name: "delete", updates: []clientv3.WatchResponse{{Events: []*clientv3.Event{{Type: clientv3.EventTypeDelete, Kv: &mvccpb.KeyValue{Key: []byte("service/A")}}}}}, retry: []time.Duration{100, 200}},
		{name: "progress", updates: []clientv3.WatchResponse{{Header: etcdserverpb.ResponseHeader{Revision: 8}}}, retry: []time.Duration{100, 200}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				client := clientv3.NewCtxClient(ctx)
				var attempts []time.Time
				client.KV = resolverKV{get: func(context.Context) (*clientv3.GetResponse, error) {
					attempts = append(attempts, time.Now())
					if len(attempts) == 10 {
						cancel()
						return nil, context.Canceled
					}
					return &clientv3.GetResponse{Header: &etcdserverpb.ResponseHeader{Revision: 7}}, nil
				}}
				client.Watcher = resolverWatch{watch: func(context.Context, int64) clientv3.WatchChan {
					updates := make(chan clientv3.WatchResponse, len(scenario.updates))
					if len(attempts) == 8 {
						for _, update := range scenario.updates {
							updates <- update
						}
					}
					close(updates)
					return updates
				}}
				conn := resolverConn{states: make(chan resolver.State, 16), errors: make(chan error, 16)}
				watcher := endpointResolver{client: client, prefix: "service/", conn: conn, done: make(chan struct{})}

				watcher.run(ctx)

				want := []time.Duration{100, 200, 400, 800, 1600, 3000, 3000}
				if scenario.retry == nil {
					want = append(want, 3000, 3000)
				} else {
					want = append(want, scenario.retry...)
				}
				var delays []time.Duration
				for index := 1; index < len(attempts); index++ {
					delays = append(delays, attempts[index].Sub(attempts[index-1])/time.Millisecond)
				}
				require.Equal(t, want, delays)
			})
		})
	}
}
