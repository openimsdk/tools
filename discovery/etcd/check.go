package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/openimsdk/tools/errs"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Check verifies if etcd is running by checking the existence of the root node and optionally creates it with a lease
func Check(ctx context.Context, etcdServers []string, etcdRoot string, createIfNotExist bool, options ...CfgOption) error {
	cfg := clientv3.Config{
		Endpoints: etcdServers,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	client, err := clientv3.New(cfg)
	if err != nil {
		return errs.WrapMsg(err, "failed to connect to etcd")
	}
	defer client.Close()

	var opCtx context.Context
	var cancel context.CancelFunc
	if cfg.DialTimeout != 0 {
		opCtx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
	} else {
		opCtx, cancel = context.WithTimeout(ctx, 10*time.Second)
	}
	defer cancel()

	resp, err := client.Get(opCtx, etcdRoot)
	if err != nil {
		return errs.WrapMsg(err, "failed to get the root node from etcd")
	}

	if len(resp.Kvs) == 0 {
		if createIfNotExist {
			var leaseTTL int64 = 10
			var leaseResp *clientv3.LeaseGrantResponse
			leaseResp, err = client.Grant(opCtx, leaseTTL)
			if err != nil {
				return errs.WrapMsg(err, "failed to create lease in etcd")
			}
			var putOpts []clientv3.OpOption
			if leaseResp != nil {
				putOpts = append(putOpts, clientv3.WithLease(leaseResp.ID))
			}

			_, err := client.Put(opCtx, etcdRoot, "", putOpts...)
			if err != nil {
				return errs.WrapMsg(err, "failed to create the root node in etcd")
			}
		} else {
			return fmt.Errorf("root node %s does not exist in etcd", etcdRoot)
		}
	}
	return nil
}
