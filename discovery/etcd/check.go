package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/utils/datautil"
)

// Check verifies if etcd is running by checking the existence of the root node and optionally creates it with a lease
func Check(ctx context.Context, etcdServers []string, etcdRoot string, createIfNotExist bool, options ...CfgOption) error {
	cfg := clientv3.Config{Endpoints: etcdServers}
	datautil.Foreach(options, func(option CfgOption) { option(&cfg) })

	client, err := clientv3.New(cfg)
	if err != nil {
		return errs.WrapMsg(err, "failed to connect to etcd")
	}
	defer client.Close()

	timeout := datautil.If(cfg.DialTimeout > 0, cfg.DialTimeout, 5*time.Second)
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if _, err := client.MemberList(opCtx); err != nil {
		return errs.WrapMsg(err, "etcd cluster unreachable")
	}

	if createIfNotExist {
		lease, err := client.Grant(opCtx, 5)
		if err != nil {
			return errs.WrapMsg(err, "failed to grant probe lease")
		}
		probeKey := datautil.If(etcdRoot != "", etcdRoot, "/_probe_health")
		if _, err := client.Put(opCtx, probeKey, "ok", clientv3.WithLease(lease.ID)); err != nil {
			return errs.WrapMsg(err, "failed to write probe key")
		}
	}

	return nil
}
