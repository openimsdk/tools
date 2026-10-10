package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc/resolver"

	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	"github.com/openimsdk/tools/utils/network"
)

type serviceWatcher struct {
	client *clientv3.Client
	prefix string
	cancel context.CancelFunc
	done   chan struct{}
	ready  chan struct{}
	once   sync.Once

	mu        sync.RWMutex
	listeners map[uint64]func([]resolver.Address, error)
	addresses []resolver.Address
	nextID    uint64
}

func newServiceWatcher(client *clientv3.Client, prefix string) *serviceWatcher {
	ctx, cancel := context.WithCancel(client.Ctx())
	watcher := &serviceWatcher{
		client:    client,
		prefix:    prefix,
		cancel:    cancel,
		done:      make(chan struct{}),
		ready:     make(chan struct{}),
		listeners: make(map[uint64]func([]resolver.Address, error)),
	}
	go watcher.run(ctx)
	return watcher
}

func (w *serviceWatcher) addListener(consume func([]resolver.Address, error)) uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.nextID++
	w.listeners[w.nextID] = consume
	if w.addresses != nil {
		consume(w.addresses, nil)
	}
	return w.nextID
}

func (w *serviceWatcher) removeListener(id uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.listeners, id)
}

func (w *serviceWatcher) getAddresses() []resolver.Address {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return datautil.CopySlice(w.addresses)
}

func (w *serviceWatcher) waitReady(ctx context.Context) error {
	select {
	case <-w.ready:
		return nil
	case <-w.done:
		return errors.New("etcd watcher: closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *serviceWatcher) close() {
	w.cancel()
	<-w.done
}

func (w *serviceWatcher) reportError(err error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, consume := range w.listeners {
		consume(nil, err)
	}
}

func (w *serviceWatcher) publish(addresses map[string]resolver.Address) {
	keys := datautil.Keys(addresses)
	sort.Strings(keys)
	addrs := datautil.Slice(keys, func(key string) resolver.Address {
		return addresses[key]
	})
	w.mu.Lock()
	w.addresses = addrs
	for _, consume := range w.listeners {
		consume(addrs, nil)
	}
	w.mu.Unlock()
	w.once.Do(func() { close(w.ready) })
}

func (w *serviceWatcher) run(ctx context.Context) {
	defer close(w.done)
	delay := 100 * time.Millisecond
	for ctx.Err() == nil {
		err := w.watchSnapshot(ctx, func() { delay = 100 * time.Millisecond })
		if ctx.Err() != nil {
			return
		}
		w.reportError(err)
		log.ZWarn(ctx, "etcd watcher failed, retrying", err, "prefix", w.prefix, "retryDelay", delay)
		if !sleepWithContext(ctx, delay) {
			return
		}
		delay = min(delay*2, 3*time.Second)
	}
}

func (w *serviceWatcher) watchSnapshot(parent context.Context, recovered func()) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	getCtx, getCancel := context.WithTimeout(ctx, 5*time.Second)
	snapshot, err := w.client.Get(getCtx, w.prefix, clientv3.WithPrefix())
	getCancel()
	if err != nil {
		return err
	}
	addresses := make(map[string]resolver.Address, len(snapshot.Kvs))
	for _, entry := range snapshot.Kvs {
		w.setAddress(addresses, string(entry.Key), entry.Value)
	}
	w.publish(addresses)
	updates := w.client.Watch(ctx, w.prefix, clientv3.WithPrefix(), clientv3.WithRev(snapshot.Header.Revision+1), clientv3.WithProgressNotify())
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, open := <-updates:
			if !open {
				return errors.New("etcd watcher: watch closed")
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
					if event.Kv != nil {
						w.setAddress(addresses, string(event.Kv.Key), event.Kv.Value)
					}
				case clientv3.EventTypeDelete:
					if event.Kv != nil {
						delete(addresses, string(event.Kv.Key))
					} else if event.PrevKv != nil {
						delete(addresses, string(event.PrevKv.Key))
					}
				}
			}
			if len(update.Events) > 0 {
				w.publish(addresses)
			}
		}
	}
}

func (w *serviceWatcher) setAddress(addresses map[string]resolver.Address, key string, value []byte) {
	var endpoint endpoints.Endpoint
	if err := json.Unmarshal(value, &endpoint); err == nil && network.IsValidHostPort(endpoint.Addr) {
		addresses[key] = resolver.Address{Addr: endpoint.Addr, Metadata: endpoint.Metadata}
		return
	}
	if lastSlash := strings.LastIndex(key, "/"); lastSlash != -1 && lastSlash < len(key)-1 {
		addr := key[lastSlash+1:]
		if network.IsValidHostPort(addr) {
			addresses[key] = resolver.Address{Addr: addr}
			return
		}
	}
	err := fmt.Errorf("invalid endpoint value at %s: %s", key, string(value))
	log.ZWarn(context.Background(), "invalid endpoint in etcd", err, "key", key, "prefix", w.prefix)
	w.reportError(err)
}
