package etcd

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

func TestGetConnReusesServiceConnection(t *testing.T) {
	// Given: a pool with a live etcd client and a service dial option.
	client, err := clientv3.New(clientv3.Config{Endpoints: []string{"127.0.0.1:1"}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	pool := newConnPool(client, "test", nil)
	defer pool.close()

	// When: the same service connection is requested twice.
	first, err := pool.GetConn(context.Background(), "service", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	second, err := pool.GetConn(context.Background(), "service", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	// Then: the pool returns the existing connection.
	if first != second {
		t.Fatal("GetConn created a second connection for the same service")
	}
}

func TestServiceWatchUpdatesPoolConnections(t *testing.T) {
	endpoint := os.Getenv("ETCD_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set ETCD_TEST_ENDPOINT to run against a local etcd instance")
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{endpoint},
		Username:  os.Getenv("ETCD_TEST_USER"),
		Password:  os.Getenv("ETCD_TEST_PASSWORD"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	root := fmt.Sprintf("pool-test-%d", time.Now().UnixNano())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	healthgrpc.RegisterHealthServer(server, health.NewServer())
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	pool := newConnPool(client, root, nil)
	defer pool.close()
	key := root + "/service/" + listener.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() { _, _ = client.Delete(context.Background(), key) }()

	// Given: a service with no registered endpoints.
	if _, err := pool.GetConns(ctx, "service", grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		t.Fatal(err)
	}
	// When: an endpoint is registered in etcd.
	if _, err := client.Put(ctx, key, fmt.Sprintf(`{"Addr":%q}`, listener.Addr().String())); err != nil {
		t.Fatal(err)
	}
	// Then: GetConns exposes the watched endpoint using one shared connection.
	for {
		conns, err := pool.GetConns(ctx, "service")
		if err != nil {
			t.Fatal(err)
		}
		if len(conns) == 1 {
			second, err := pool.GetConns(ctx, "service")
			if err != nil || len(second) != 1 || conns[0] != second[0] {
				t.Fatalf("connection was not reused: %v, %v, %v", conns, second, err)
			}
			serviceConn, err := pool.GetConn(ctx, "service", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := healthgrpc.NewHealthClient(serviceConn).Check(ctx, &healthgrpc.HealthCheckRequest{}); err != nil {
				t.Fatalf("GetConn resolver did not forward the watched address: %v", err)
			}
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}
