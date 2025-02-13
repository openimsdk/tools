package standalone

import (
	"context"
	"sync"

	"github.com/openimsdk/tools/discovery"
)

type keyValue struct {
	lock sync.RWMutex
	kv   map[string][]byte
}

func (x *keyValue) SetKey(ctx context.Context, key string, data []byte) error {
	tmp := make([]byte, len(data))
	copy(tmp, data)
	x.lock.Lock()
	if x.kv == nil {
		x.kv = make(map[string][]byte)
	}
	x.kv[key] = tmp
	x.lock.Unlock()
	return nil
}

func (x *keyValue) GetKey(ctx context.Context, key string) ([]byte, error) {
	x.lock.RLock()
	defer x.lock.RUnlock()
	if x.kv != nil {
		if v, ok := x.kv[key]; ok {
			tmp := make([]byte, len(v))
			copy(tmp, v)
			return tmp, nil
		}
	}
	return nil, nil
}

func (x *keyValue) DelData(ctx context.Context, key string) error {
	x.lock.Lock()
	if x.kv != nil {
		delete(x.kv, key)
	}
	x.lock.Unlock()
	return nil
}

func (x *keyValue) WatchKey(ctx context.Context, key string, fn discovery.WatchKeyHandler) error {
	return discovery.ErrNotSupportedKeyValue
}
