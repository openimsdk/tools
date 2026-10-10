package inprocess

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/openimsdk/tools/discovery"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMain(m *testing.M) {
	code := m.Run()
	_ = os.RemoveAll("logs")
	os.Exit(code)
}

func TestServiceRegistryRegisterAndConns(t *testing.T) {
	ctx := context.Background()
	registry := newServiceRegistry()

	require.NoError(t, registry.Register(ctx, "messagegateway", "127.0.0.1", 10001))
	require.Equal(t, "127.0.0.1:10001", registry.Target())

	registry.setBroadcastAddressProvider("secret", func(context.Context) ([]string, error) {
		return []string{
			"127.0.0.1:10001",
			" ",
			"127.0.0.1:10002",
			"127.0.0.1:10002",
		}, nil
	})

	conns, err := registry.GetConns(ctx, "messagegateway")
	require.NoError(t, err)
	require.Len(t, conns, 2)
	require.True(t, registry.IsSelfNode(conns[0]))
	require.False(t, registry.IsSelfNode(conns[1]))

	type targetConn interface {
		Target() string
		grpc.ClientConnInterface
	}
	remote, ok := conns[1].(targetConn)
	require.True(t, ok)
	require.Equal(t, "127.0.0.1:10002", remote.Target())
}

func TestServiceRegistryRegisterValidation(t *testing.T) {
	registry := newServiceRegistry()
	ctx := context.Background()

	require.Error(t, registry.Register(ctx, " ", "127.0.0.1", 10001))
	require.Error(t, registry.Register(ctx, "messagegateway", " ", 10001))
	require.Error(t, registry.Register(ctx, "messagegateway", "127.0.0.1", 0))
	require.Error(t, registry.Register(ctx, "messagegateway", "127.0.0.1", 65536))
}

func TestServiceSpecificTargetFiltersSelfNode(t *testing.T) {
	ctx := context.Background()
	registry := newServiceRegistry()

	require.NoError(t, registry.Register(ctx, "service-a", "127.0.0.1", 10001))
	require.NoError(t, registry.Register(ctx, "messagegateway", "127.0.0.1", 10002))
	registry.setBroadcastAddressProvider("secret", func(context.Context) ([]string, error) {
		return []string{
			"127.0.0.1:10001",
			"127.0.0.1:10002",
			"127.0.0.1:10003",
		}, nil
	})

	conns, err := registry.GetConns(ctx, "messagegateway")
	require.NoError(t, err)
	require.Len(t, conns, 3)

	targets := make([]string, 0, len(conns)-1)
	for _, conn := range conns[1:] {
		targets = append(targets, conn.(interface{ Target() string }).Target())
	}
	require.ElementsMatch(t, []string{"127.0.0.1:10001", "127.0.0.1:10003"}, targets)
}

func TestMethodWaiterReleasedOnContextCancel(t *testing.T) {
	methods := newMethodRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := methods.getUnaryHandler(ctx, "/inprocess.Test/Missing")
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, methods.waiters)
}

func TestCloseWakesMethodWaiter(t *testing.T) {
	registry := newServiceRegistry()
	errCh := make(chan error, 1)

	go func() {
		var reply wrapperspb.StringValue
		conn, err := registry.GetConn(context.Background(), "inprocess.Test")
		if err != nil {
			errCh <- err
			return
		}
		errCh <- conn.Invoke(context.Background(), "/inprocess.Test/Missing", wrapperspb.String("hello"), &reply)
	}()

	require.Eventually(t, func() bool {
		registry.connManager.local.methods.mu.Lock()
		defer registry.connManager.local.methods.mu.Unlock()
		return len(registry.connManager.local.methods.waiters) == 1
	}, time.Second, time.Millisecond)

	registry.Close()

	select {
	case err := <-errCh:
		require.Error(t, err)
		require.Contains(t, err.Error(), "internal error")
	case <-time.After(time.Second):
		t.Fatal("missing method waiter was not woken by Close")
	}
}

func TestParseRemoteAddress(t *testing.T) {
	endpoint, target := parseRemoteAddress("127.0.0.1:10001")
	require.Equal(t, "http://127.0.0.1:10001"+BroadcastPath, endpoint)
	require.Equal(t, "127.0.0.1:10001", target)

	endpoint, target = parseRemoteAddress("https://example.com:443/")
	require.Equal(t, "https://example.com:443"+BroadcastPath, endpoint)
	require.Equal(t, "example.com:443", target)
}

func TestRegistryInvokesLocalUnaryService(t *testing.T) {
	ctx := context.Background()
	registry := newServiceRegistry()
	registry.RegisterService(&grpc.ServiceDesc{
		ServiceName: "inprocess.Test",
		HandlerType: (*testUnaryService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Echo",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					req := new(wrapperspb.StringValue)
					if err := dec(req); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(testUnaryService).Echo(ctx, req)
					}
					info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/inprocess.Test/Echo"}
					handler := func(ctx context.Context, in any) (any, error) {
						return srv.(testUnaryService).Echo(ctx, in.(*wrapperspb.StringValue))
					}
					return interceptor(ctx, req, info, handler)
				},
			},
		},
	}, testUnaryServer{})

	conn, err := registry.GetConn(ctx, "inprocess.Test")
	require.NoError(t, err)

	var reply wrapperspb.StringValue
	err = conn.Invoke(ctx, "/inprocess.Test/Echo", wrapperspb.String("hello"), &reply)
	require.NoError(t, err)
	require.Equal(t, "hello:ok", reply.Value)
}

func TestKeyValuePrefixCloneWatchAndLease(t *testing.T) {
	ctx := context.Background()
	kv := &memoryKV{}

	data := []byte("alpha")
	require.NoError(t, kv.SetKey(ctx, "prometheus/a/0", data))
	data[0] = 'x'

	got, err := kv.GetKey(ctx, "prometheus/a/0")
	require.NoError(t, err)
	require.Equal(t, []byte("alpha"), got)

	got[0] = 'y'
	got, err = kv.GetKey(ctx, "prometheus/a/0")
	require.NoError(t, err)
	require.Equal(t, []byte("alpha"), got)

	require.NoError(t, kv.SetKey(ctx, "prometheus/a/1", []byte("beta")))
	require.NoError(t, kv.SetKey(ctx, "prometheus/b/0", []byte("other")))
	values, err := kv.GetKeyWithPrefix(ctx, "prometheus/a")
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("alpha"), []byte("beta")}, values)

	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := make(chan *discovery.WatchKey, 4)
	errCh := make(chan error, 1)
	go func() {
		errCh <- kv.WatchKey(watchCtx, "lease/", func(event *discovery.WatchKey) error {
			events <- event
			return nil
		})
	}()
	require.Eventually(t, func() bool {
		kv.mu.RLock()
		defer kv.mu.RUnlock()
		return len(kv.subscribers) == 1
	}, time.Second, time.Millisecond)

	require.NoError(t, kv.setValue(ctx, "lease/key", []byte("temp"), 20*time.Millisecond))
	requireWatchEvent(t, events, discovery.WatchTypePut, "lease/key", []byte("temp"))
	requireWatchEvent(t, events, discovery.WatchTypeDelete, "lease/key", nil)

	require.Eventually(t, func() bool {
		got, err := kv.GetKey(ctx, "lease/key")
		require.NoError(t, err)
		return got == nil
	}, time.Second, time.Millisecond)

	cancel()
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("watch did not stop")
	}
}

func TestWatchSlowSubscriberDoesNotBlockWriters(t *testing.T) {
	ctx := context.Background()
	kv := &memoryKV{}

	block := make(chan struct{})
	var mu sync.Mutex
	var got []int
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		first := true
		errCh <- kv.WatchKey(watchCtx, "order/", func(event *discovery.WatchKey) error {
			if first {
				first = false
				<-block // stall the handler while writers keep going
			}
			value, err := strconv.Atoi(string(event.Value))
			if err != nil {
				return err
			}
			mu.Lock()
			got = append(got, value)
			mu.Unlock()
			return nil
		})
	}()
	require.Eventually(t, func() bool {
		kv.mu.RLock()
		defer kv.mu.RUnlock()
		return len(kv.subscribers) == 1
	}, time.Second, time.Millisecond)

	const total = 100
	writesDone := make(chan struct{})
	go func() {
		defer close(writesDone)
		for i := 0; i < total; i++ {
			_ = kv.SetKey(ctx, "order/key", []byte(strconv.Itoa(i)))
		}
	}()
	select {
	case <-writesDone:
	case <-time.After(time.Second):
		t.Fatal("writers blocked by a stalled watch handler")
	}
	close(block)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == total
	}, time.Second, time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	for i, value := range got {
		require.Equal(t, i, value, "watch events delivered out of order")
	}

	cancel()
	require.ErrorIs(t, <-errCh, context.Canceled)
}

type testUnaryService interface {
	Echo(context.Context, *wrapperspb.StringValue) (*wrapperspb.StringValue, error)
}

type testUnaryServer struct{}

func (testUnaryServer) Echo(_ context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String(req.Value + ":ok"), nil
}

func requireWatchEvent(t *testing.T, events <-chan *discovery.WatchKey, typ discovery.WatchType, key string, value []byte) {
	t.Helper()

	select {
	case event := <-events:
		require.Equal(t, typ, event.Type)
		require.Equal(t, []byte(key), event.Key)
		require.Equal(t, value, event.Value)
	case <-time.After(time.Second):
		t.Fatalf("watch event %s for %s not received", typ, key)
	}
}
