package etcd

import (
	"context"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/utils/datautil"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// keepAliveLease maintains the lease alive by sending keep-alive requests
func (r *SvcDiscoveryRegistryImpl) keepAliveLease(leaseID clientv3.LeaseID) {
	ch, err := r.client.KeepAlive(context.Background(), leaseID)
	if err != nil {
		return
	}
	for ka := range ch {
		if ka == nil {
			return
		}
	}
}

func (r *SvcDiscoveryRegistryImpl) SetKey(ctx context.Context, key string, data []byte) error {
	if _, err := r.client.Put(ctx, key, string(data)); err != nil {
		return errs.WrapMsg(err, "etcd put err")
	}
	return nil
}

func (r *SvcDiscoveryRegistryImpl) SetWithLease(ctx context.Context, key string, val []byte, ttl int64) error {
	leaseResp, err := r.client.Grant(ctx, ttl) //
	if err != nil {
		return errs.Wrap(err)
	}

	_, err = r.client.Put(ctx, key, string(val), clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return errs.Wrap(err)
	}

	go r.keepAliveLease(leaseResp.ID)

	return nil
}

func (r *SvcDiscoveryRegistryImpl) GetKey(ctx context.Context, key string) ([]byte, error) {
	resp, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, errs.WrapMsg(err, "etcd get err")
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	return resp.Kvs[0].Value, nil
}

func (r *SvcDiscoveryRegistryImpl) GetKeyWithPrefix(ctx context.Context, key string) ([][]byte, error) {
	resp, err := r.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, errs.WrapMsg(err, "etcd get err")
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	return datautil.Batch(func(kv *mvccpb.KeyValue) []byte { return kv.Value }, resp.Kvs), nil
}

func (r *SvcDiscoveryRegistryImpl) DelData(ctx context.Context, key string) error {
	if _, err := r.client.Delete(ctx, key); err != nil {
		return errs.WrapMsg(err, "etcd delete err")
	}
	return nil
}
