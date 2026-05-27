package etcd

import (
	"context"
	"fmt"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

func (r *SvcDiscoveryRegistryImpl) combineKeyWithPrefix(key string) string {
	return fmt.Sprintf("%s/%s", r.rootDirectory, key)
}

// keepAliveLease maintains the lease alive by sending keep-alive requests
func (r *SvcDiscoveryRegistryImpl) keepAliveLease(ctx context.Context, leaseID clientv3.LeaseID) {
	ch, err := r.client.KeepAlive(ctx, leaseID)
	if err != nil {
		return
	}
	for ka := range ch {
		if ka == nil {
			return
		}
	}
}

func (r *SvcDiscoveryRegistryImpl) newKVKeepAliveContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	r.kvKeepAliveMu.Lock()
	r.kvKeepAliveCancel = append(r.kvKeepAliveCancel, cancel)
	r.kvKeepAliveMu.Unlock()
	return ctx
}

func (r *SvcDiscoveryRegistryImpl) stopKVKeepAlives() {
	r.kvKeepAliveMu.Lock()
	cancels := r.kvKeepAliveCancel
	r.kvKeepAliveCancel = nil
	r.kvKeepAliveMu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

func (r *SvcDiscoveryRegistryImpl) SetKey(ctx context.Context, key string, data []byte) error {
	if _, err := r.client.Put(ctx, r.combineKeyWithPrefix(key), string(data)); err != nil {
		return errs.WrapMsg(err, "etcd put err")
	}
	return nil
}

func (r *SvcDiscoveryRegistryImpl) setKeyWithLease(ctx context.Context, key string, val []byte, ttl int64) (clientv3.LeaseID, error) {
	leaseResp, err := r.client.Grant(ctx, ttl) //
	if err != nil {
		return 0, errs.Wrap(err)
	}

	_, err = r.client.Put(ctx, r.combineKeyWithPrefix(key), string(val), clientv3.WithLease(leaseResp.ID))
	if err != nil {
		_, _ = r.client.Revoke(ctx, leaseResp.ID)
		return 0, errs.Wrap(err)
	}

	return leaseResp.ID, nil
}

func (r *SvcDiscoveryRegistryImpl) SetWithLease(ctx context.Context, key string, val []byte, ttl int64) error {
	id, err := r.setKeyWithLease(ctx, key, val, ttl)
	if err != nil {
		return errs.Wrap(err)
	}
	keepCtx := r.newKVKeepAliveContext()

	go func() {
		for {
			r.keepAliveLease(keepCtx, id)
			if keepCtx.Err() != nil {
				return
			}

			log.ZWarn(
				context.Background(),
				"etcd lease keepalive stopped, resetting key with lease",
				nil,
				zap.String("key", key),
				zap.Int64("leaseID", int64(id)),
			)

			if !sleepWithContext(keepCtx, keepAliveRetryDelay) {
				return
			}

			retryCtx, cancel := withTimeout(keepCtx, defaultRegisterTimeout)
			newID, err := r.setKeyWithLease(retryCtx, key, val, ttl)
			cancel()
			if err != nil {
				log.ZWarn(
					context.Background(),
					"reset etcd key with lease failed",
					err,
					zap.String("key", key),
					zap.Int64("leaseID", int64(id)),
				)
				continue
			}
			id = newID
		}
	}()

	return nil
}

func (r *SvcDiscoveryRegistryImpl) GetKey(ctx context.Context, key string) ([]byte, error) {
	resp, err := r.client.Get(ctx, r.combineKeyWithPrefix(key))
	if err != nil {
		return nil, errs.WrapMsg(err, "etcd get err")
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	return resp.Kvs[0].Value, nil
}

func (r *SvcDiscoveryRegistryImpl) GetKeyWithPrefix(ctx context.Context, key string) ([][]byte, error) {
	resp, err := r.client.Get(ctx, r.combineKeyWithPrefix(key), clientv3.WithPrefix())
	if err != nil {
		return nil, errs.WrapMsg(err, "etcd get err")
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	return datautil.Batch(func(kv *mvccpb.KeyValue) []byte { return kv.Value }, resp.Kvs), nil
}

func (r *SvcDiscoveryRegistryImpl) DelData(ctx context.Context, key string) error {
	if _, err := r.client.Delete(ctx, r.combineKeyWithPrefix(key)); err != nil {
		return errs.WrapMsg(err, "etcd delete err")
	}
	return nil
}
