package etcd

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/log"
)

type watchKeyEntry struct {
	key    string
	ctx    context.Context
	cancel context.CancelFunc

	mu   sync.RWMutex
	subs map[*watchKeySubscriber]struct{}
}

type watchKeySubscriber struct {
	ctx    context.Context
	cancel context.CancelFunc
	events chan *discovery.WatchKey
}

func (e *watchKeyEntry) addSubscriber(sub *watchKeySubscriber) {
	e.mu.Lock()
	if e.subs == nil {
		e.subs = make(map[*watchKeySubscriber]struct{})
	}
	e.subs[sub] = struct{}{}
	e.mu.Unlock()
}

func (e *watchKeyEntry) removeSubscriber(sub *watchKeySubscriber) bool {
	e.mu.Lock()
	if e.subs == nil {
		e.mu.Unlock()
		return true
	}
	delete(e.subs, sub)
	empty := len(e.subs) == 0
	e.mu.Unlock()
	return empty
}

func (e *watchKeyEntry) broadcast(h *kvHub, event *discovery.WatchKey) {
	e.mu.RLock()
	if len(e.subs) == 0 {
		e.mu.RUnlock()
		return
	}
	subs := make([]*watchKeySubscriber, 0, len(e.subs))
	for sub := range e.subs {
		subs = append(subs, sub)
	}
	e.mu.RUnlock()

	for _, sub := range subs {
		if !sub.push(event) {
			h.removeWatchKeySubscriber(e.key, e, sub)
		}
	}
}

func (e *watchKeyEntry) closeSubscribers() {
	e.mu.RLock()
	if len(e.subs) == 0 {
		e.mu.RUnlock()
		return
	}
	subs := make([]*watchKeySubscriber, 0, len(e.subs))
	for sub := range e.subs {
		subs = append(subs, sub)
	}
	e.mu.RUnlock()

	for _, sub := range subs {
		sub.cancel()
	}
}

func (s *watchKeySubscriber) push(event *discovery.WatchKey) bool {
	select {
	case <-s.ctx.Done():
		return false
	default:
	}

	select {
	case s.events <- event:
		return true
	case <-s.ctx.Done():
		return false
	}
}

func (e *watchKeyEntry) run(h *kvHub) {
	var sdkWatcher clientv3.Watcher
	var ownedWatcher clientv3.Watcher
	defer func() {
		if ownedWatcher != nil {
			_ = ownedWatcher.Close()
		}
	}()
	defer func() {
		e.closeSubscribers()
		h.removeWatchKeyEntry(e.key, e)
	}()

	delay := 100 * time.Millisecond
	for {
		select {
		case <-e.ctx.Done():
			return
		default:
		}

		client := h.client
		if client == nil {
			return
		}
		if sdkWatcher == nil {
			sdkWatcher = client.Watcher
		}
		var watchChan clientv3.WatchChan

		getCtx, getCancel := context.WithTimeout(e.ctx, 5*time.Second)
		_, err := client.Get(getCtx, e.key, clientv3.WithKeysOnly(), clientv3.WithLimit(1))
		getCancel()
		if err != nil {
			log.ZWarn(e.ctx, "watch key snapshot err", err, zap.String("key", e.key))
			goto reconnect
		}

		if e.ctx.Err() != nil {
			return
		}

		watchChan = sdkWatcher.Watch(e.ctx, e.key, clientv3.WithPrefix())
		for {
			select {
			case <-e.ctx.Done():
				return
			case resp, ok := <-watchChan:
				if !ok {
					goto reconnect
				}
				if err := resp.Err(); err != nil {
					err = rpctypes.Error(err)
					if errors.Is(err, rpctypes.ErrInvalidAuthToken) || errors.Is(err, rpctypes.ErrAuthOldRevision) {
						if ownedWatcher != nil {
							_ = ownedWatcher.Close()
						}
						ownedWatcher = clientv3.NewWatcher(client)
						sdkWatcher = ownedWatcher
					}
					log.ZWarn(context.Background(), "watch key resp err", err, zap.String("key", e.key))
					goto reconnect
				}
				delay = 100 * time.Millisecond
				for _, event := range resp.Events {
					watchKey := &discovery.WatchKey{Key: event.Kv.Key, Value: event.Kv.Value}
					switch event.Type {
					case mvccpb.PUT:
						watchKey.Type = discovery.WatchTypePut
					case mvccpb.DELETE:
						watchKey.Type = discovery.WatchTypeDelete
					default:
						continue
					}
					e.broadcast(h, watchKey)
				}
			}
		}

	reconnect:
		if !sleepWithContext(e.ctx, delay) {
			return
		}
		delay = min(delay*2, 3*time.Second)
	}
}

func (h *kvHub) stopKeyWatches() {
	h.watchMu.Lock()
	entries := make([]*watchKeyEntry, 0, len(h.watchEntries))
	for _, entry := range h.watchEntries {
		entries = append(entries, entry)
	}
	h.watchEntries = make(map[string]*watchKeyEntry)
	h.watchMu.Unlock()

	for _, entry := range entries {
		if entry.cancel != nil {
			entry.cancel()
		}
	}
}

func (h *kvHub) getOrCreateWatchKeyEntry(key string) (*watchKeyEntry, error) {
	h.watchMu.Lock()
	if entry, ok := h.watchEntries[key]; ok {
		h.watchMu.Unlock()
		return entry, nil
	}
	if h.client == nil {
		h.watchMu.Unlock()
		return nil, fmt.Errorf("etcd client closed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry := &watchKeyEntry{
		key:    key,
		ctx:    ctx,
		cancel: cancel,
		subs:   make(map[*watchKeySubscriber]struct{}),
	}
	h.watchEntries[key] = entry
	h.watchMu.Unlock()

	go entry.run(h)
	return entry, nil
}

func (h *kvHub) removeWatchKeySubscriber(key string, entry *watchKeyEntry, sub *watchKeySubscriber) {
	if sub == nil || entry == nil {
		return
	}
	sub.cancel()
	empty := entry.removeSubscriber(sub)
	if !empty {
		return
	}

	h.watchMu.Lock()
	if current, ok := h.watchEntries[key]; ok && current == entry {
		delete(h.watchEntries, key)
	}
	h.watchMu.Unlock()

	if entry.cancel != nil {
		entry.cancel()
	}
}

func (h *kvHub) removeWatchKeyEntry(key string, entry *watchKeyEntry) {
	h.watchMu.Lock()
	if current, ok := h.watchEntries[key]; ok && current == entry {
		delete(h.watchEntries, key)
	}
	h.watchMu.Unlock()
}

func (h *kvHub) WatchKey(ctx context.Context, key string, fn discovery.WatchKeyHandler) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if fn == nil {
		return fmt.Errorf("watch handler is nil")
	}

	key = h.combineKeyWithPrefix(key)

	entry, err := h.getOrCreateWatchKeyEntry(key)
	if err != nil {
		return err
	}

	subCtx, cancel := context.WithCancel(ctx)
	sub := &watchKeySubscriber{
		ctx:    subCtx,
		cancel: cancel,
		events: make(chan *discovery.WatchKey, 16),
	}

	entry.addSubscriber(sub)
	defer h.removeWatchKeySubscriber(key, entry, sub)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sub.ctx.Done():
			return nil
		case event := <-sub.events:
			if event == nil {
				continue
			}
			if err := fn(event); err != nil {
				return err
			}
		}
	}
}

func (h *kvHub) close() {
	h.stopKeyWatches()
	h.stopKVKeepAlives()
}
