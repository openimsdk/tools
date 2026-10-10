package etcd

import (
	"context"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// reconnectingWatcher is a transparent wrapper around clientv3.Watcher that automatically
// recovers from stream-level failures, specifically authentication token expiration.
//
// Background:
// In etcd clientv3, watch operations multiplex over shared gRPC streams. Authentication
// tokens (such as short-lived JWTs) are only refreshed on unary RPC retries or when establishing
// a brand-new stream. An active watch stream never refreshes tokens in-place, and clientv3 exposes
// no public API on Watcher to force stream recreation or token refresh. Once a token expires,
// the stream halts with "etcdserver: invalid auth token", and watches reusing that stream become
// trapped in an infinite retry loop.
//
// Solution:
// Whenever a watch channel closes or yields an error, reconnectingWatcher replaces the underlying
// clientv3.Watcher with a fresh instance via clientv3.NewWatcher(client). Creating a new Watcher
// forces a new gRPC stream, which triggers the client's stream interceptor to fetch a fresh auth
// token and restores connectivity transparently to callers.
type reconnectingWatcher struct {
	client *clientv3.Client

	mu      sync.Mutex
	watcher clientv3.Watcher
	closed  bool
}

func newReconnectingWatcher(client *clientv3.Client, watch clientv3.Watcher) *reconnectingWatcher {
	return &reconnectingWatcher{client: client, watcher: watch}
}

func (w *reconnectingWatcher) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	out := make(chan clientv3.WatchResponse)
	go func() {
		defer close(out)
		watch := w.current()
		if watch == nil {
			return
		}
		updates := watch.Watch(ctx, key, opts...)
		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-updates:
				if !ok {
					w.replace(watch)
					return
				}
				if err := update.Err(); err != nil {
					w.replace(watch)
				}
				select {
				case out <- update:
				case <-ctx.Done():
					return
				}
				if update.Err() != nil {
					return
				}
			}
		}
	}()
	return out
}

func (w *reconnectingWatcher) RequestProgress(ctx context.Context) error {
	watch := w.current()
	if watch == nil {
		return context.Canceled
	}
	return watch.RequestProgress(ctx)
}

func (w *reconnectingWatcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	watch := w.watcher
	w.watcher = nil
	w.mu.Unlock()
	if watch == nil {
		return nil
	}
	return watch.Close()
}

func (w *reconnectingWatcher) current() clientv3.Watcher {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	return w.watcher
}

func (w *reconnectingWatcher) replace(old clientv3.Watcher) {
	w.mu.Lock()
	if w.closed || w.watcher != old {
		w.mu.Unlock()
		return
	}
	w.watcher = clientv3.NewWatcher(w.client)
	w.mu.Unlock()
	_ = old.Close()
}
