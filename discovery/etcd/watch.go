package etcd

import (
	"context"
	"fmt"
	"sync"

	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/log"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
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

func (e *watchKeyEntry) broadcast(r *SvcDiscoveryRegistryImpl, event *discovery.WatchKey) {
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
			r.removeWatchKeySubscriber(e.key, e, sub)
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

func (e *watchKeyEntry) run(r *SvcDiscoveryRegistryImpl) {
	defer func() {
		e.closeSubscribers()
		r.removeWatchKeyEntry(e.key, e)
	}()

	watchChan := r.client.Watch(e.ctx, e.key, clientv3.WithPrefix())
	for {
		select {
		case <-e.ctx.Done():
			return
		case resp, ok := <-watchChan:
			if !ok {
				return
			}
			if resp.Err() != nil {
				log.ZWarn(context.Background(), "watch key resp err", resp.Err(), zap.String("key", e.key))
				continue
			}
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
				e.broadcast(r, watchKey)
			}
		}
	}
}

// watchServiceChanges watches for changes in the service directory
func (r *SvcDiscoveryRegistryImpl) watchServiceChanges() {
	for _, s := range r.watchNames {
		if err := r.ensureServiceWatch(s); err != nil {
			log.ZWarn(context.Background(), "ensure service watch err", err, zap.String("service", s))
		}
	}
}

func (r *SvcDiscoveryRegistryImpl) ensureServiceWatch(service string) error {
	r.serviceWatchMu.Lock()
	if _, exists := r.serviceWatchers[service]; exists {
		r.serviceWatchMu.Unlock()
		return nil
	}

	if r.client == nil {
		r.serviceWatchMu.Unlock()
		return fmt.Errorf("etcd client closed")
	}

	watchCtx, cancel := context.WithCancel(context.Background())
	r.serviceWatchers[service] = cancel
	r.serviceWatchMu.Unlock()

	go r.runServiceWatch(watchCtx, service)

	return nil
}

func (r *SvcDiscoveryRegistryImpl) runServiceWatch(ctx context.Context, service string) {
	watchChan := r.client.Watch(ctx, r.combineKeyWithPrefix(service), clientv3.WithPrefix())
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-watchChan:
			if !ok {
				return
			}
			if err := r.initializeConnMap(service); err != nil {
				log.ZWarn(context.Background(), "initializeConnMap in watch err", err, zap.String("service", service))
			}
		}
	}
}

func (r *SvcDiscoveryRegistryImpl) stopServiceWatches() {
	r.serviceWatchMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(r.serviceWatchers))
	for _, cancel := range r.serviceWatchers {
		cancels = append(cancels, cancel)
	}
	r.serviceWatchers = make(map[string]context.CancelFunc)
	r.serviceWatchMu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

func (r *SvcDiscoveryRegistryImpl) stopKeyWatches() {
	r.watchKeyMu.Lock()
	entries := make([]*watchKeyEntry, 0, len(r.watchKeyEntries))
	for _, entry := range r.watchKeyEntries {
		entries = append(entries, entry)
	}
	r.watchKeyEntries = make(map[string]*watchKeyEntry)
	r.watchKeyMu.Unlock()

	for _, entry := range entries {
		if entry.cancel != nil {
			entry.cancel()
		}
	}
}

func (r *SvcDiscoveryRegistryImpl) getOrCreateWatchKeyEntry(key string) (*watchKeyEntry, error) {
	r.watchKeyMu.Lock()
	if entry, ok := r.watchKeyEntries[key]; ok {
		r.watchKeyMu.Unlock()
		return entry, nil
	}
	if r.client == nil {
		r.watchKeyMu.Unlock()
		return nil, fmt.Errorf("etcd client closed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry := &watchKeyEntry{
		key:    key,
		ctx:    ctx,
		cancel: cancel,
		subs:   make(map[*watchKeySubscriber]struct{}),
	}
	r.watchKeyEntries[key] = entry
	r.watchKeyMu.Unlock()

	go entry.run(r)
	return entry, nil
}

func (r *SvcDiscoveryRegistryImpl) removeWatchKeySubscriber(key string, entry *watchKeyEntry, sub *watchKeySubscriber) {
	if sub == nil || entry == nil {
		return
	}
	sub.cancel()
	empty := entry.removeSubscriber(sub)
	if !empty {
		return
	}

	r.watchKeyMu.Lock()
	if current, ok := r.watchKeyEntries[key]; ok && current == entry {
		delete(r.watchKeyEntries, key)
	}
	r.watchKeyMu.Unlock()

	if entry.cancel != nil {
		entry.cancel()
	}
}

func (r *SvcDiscoveryRegistryImpl) removeWatchKeyEntry(key string, entry *watchKeyEntry) {
	r.watchKeyMu.Lock()
	if current, ok := r.watchKeyEntries[key]; ok && current == entry {
		delete(r.watchKeyEntries, key)
	}
	r.watchKeyMu.Unlock()
}

func (r *SvcDiscoveryRegistryImpl) WatchKey(ctx context.Context, key string, fn discovery.WatchKeyHandler) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if fn == nil {
		return fmt.Errorf("watch handler is nil")
	}

	key = r.combineKeyWithPrefix(key)

	entry, err := r.getOrCreateWatchKeyEntry(key)
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
	defer r.removeWatchKeySubscriber(key, entry, sub)

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
