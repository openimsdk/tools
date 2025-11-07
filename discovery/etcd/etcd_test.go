package etcd

import (
	"context"
	"github.com/openimsdk/tools/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"testing"
	"time"
)

const testServerName = "auth"

func getEtcd() *SvcDiscoveryRegistryImpl {
	var endpoints []string
	endpoints = []string{"localhost:12379"}
	endpoints = []string{
		"127.0.0.1:2379",  // etcd1
		"127.0.0.1:22379", // etcd2
		"127.0.0.1:32379", // etcd3
	}
	// 127.0.0.1:2379, 127.0.0.1:22379, 127.0.0.1:32379

	var watchNames []string

	//watchNames = []string{
	//	"auth",
	//}

	r, err := NewSvcDiscoveryRegistry("openim", endpoints, watchNames)
	if err != nil {
		panic(err)
	}
	r.AddOption(grpc.WithTransportCredentials(insecure.NewCredentials()))
	return r
}

func TestGetConn(t *testing.T) {
	r := getEtcd()
	for i := 1; ; i++ {
		cs, err := r.GetConns(context.Background(), testServerName)
		if err == nil {
			t.Log("get conns success:", i, cs)
		} else {
			t.Log("get conns failed:", i, err)
		}
		time.Sleep(time.Second)
	}
}

func TestWatch(t *testing.T) {
	r := getEtcd()
	t.Log("start watch")
	for i := 0; i < 5; i++ {
		index := i + 1
		go func() {
			err := r.WatchKey(context.Background(), "test-user", func(data *discovery.WatchKey) error {
				t.Log(index, "watch data:", string(data.Key), string(data.Value), data.Type)
				return nil
			})
			t.Log(err)
		}()
	}

	for i := 1; ; i++ {
		cs, err := r.GetConn(context.Background(), testServerName)
		if err == nil {
			t.Log("get conns success:", i, cs)
		} else {
			t.Log("get conns failed:", i, err)
		}
		time.Sleep(time.Second)
	}

	select {}
}

func TestGetValue(t *testing.T) {
	r := getEtcd()
	t.Log("start watch")
	val, err := r.GetKeyWithPrefix(context.Background(), "openim/test-user")
	if err != nil {
		panic(err)
	}
	t.Log(val)
}

func TestRegister(t *testing.T) {
	r := getEtcd()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	t.Log("listening on port", port)
	if err := r.Register(context.Background(), testServerName, "192.168.10.105", port); err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		t.Log("registering...", i)
	}
	start := time.Now()
	t.Log("start close", start)
	r.Close()
	t.Log("closed", time.Since(start))
	time.Sleep(time.Second)
}

func TestWatch1(t *testing.T) {
	r := getEtcd()
	t.Log("start watch")
	err := r.WatchKey(context.Background(), testServerName, func(data *discovery.WatchKey) error {
		t.Logf("[%s] watch data key %s, value %s type %s", time.Now().Format(time.TimeOnly), string(data.Key), string(data.Value), data.Type)
		return nil
	})
	t.Log(err)
}
