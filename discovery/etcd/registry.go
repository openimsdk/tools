package etcd

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/openimsdk/tools/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Register registers a new service endpoint with etcd
func (r *SvcDiscoveryRegistryImpl) Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error {
	r.regMu.Lock()
	defer r.regMu.Unlock()

	if r.client == nil {
		return fmt.Errorf("etcd client is closed")
	}

	if r.keepAliveCancel != nil {
		r.keepAliveCancel()
		r.keepAliveCancel = nil
	}

	if r.leaseID != 0 {
		if _, err := r.client.Revoke(context.Background(), r.leaseID); err != nil {
			log.ZWarn(ctx, "failed to revoke previous lease", err, zap.String("service", serviceName), zap.String("addr", net.JoinHostPort(host, strconv.Itoa(port))))
		}
		r.leaseID = 0
	}

	registerCtx, cancel := withTimeout(ctx, defaultRegisterTimeout)
	defer cancel()

	if err := r.registerLocked(registerCtx, serviceName, host, port); err != nil {
		return err
	}

	keepCtx, keepCancel := context.WithCancel(context.Background())
	r.keepAliveCancel = keepCancel
	go r.keepAliveLoop(keepCtx)

	return nil
}

func (r *SvcDiscoveryRegistryImpl) registerLocked(ctx context.Context, serviceName, host string, port int) error {
	if ctx == nil {
		ctx = context.Background()
	}

	serviceDir := r.combineKeyWithPrefix(serviceName)
	serviceKey := fmt.Sprintf("%s/%s", serviceDir, net.JoinHostPort(host, strconv.Itoa(port)))

	manager, err := endpoints.NewManager(r.client, serviceDir)
	if err != nil {
		return err
	}

	leaseResp, err := r.client.Grant(ctx, defaultLeaseTTL)
	if err != nil {
		return err
	}

	endpointAddr := net.JoinHostPort(host, strconv.Itoa(port))
	endpoint := endpoints.Endpoint{Addr: endpointAddr}

	if err := manager.AddEndpoint(ctx, serviceKey, endpoint, clientv3.WithLease(leaseResp.ID)); err != nil {
		_, _ = r.client.Revoke(context.Background(), leaseResp.ID)
		return err
	}

	r.endpointMgr = manager
	r.serviceKey = serviceKey
	r.leaseID = leaseResp.ID
	r.rpcRegisterTarget = endpointAddr
	r.registeredService = serviceName
	r.registeredHost = host
	r.registeredPort = port

	return nil
}

func (r *SvcDiscoveryRegistryImpl) keepAliveLoop(ctx context.Context) {
outer:
	for {
		if ctx.Err() != nil {
			return
		}
		client := r.client
		if client == nil {
			return
		}

		r.regMu.Lock()
		leaseID := r.leaseID
		r.regMu.Unlock()
		if leaseID == 0 {
			return
		}

		ch, err := client.KeepAlive(ctx, leaseID)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if !r.reRegister(ctx, err) {
				if !sleepWithContext(ctx, keepAliveRetryDelay) {
					return
				}
			}
			continue
		}

		for {
			select {
			case <-ctx.Done():
				return
			case ka, ok := <-ch:
				if !ok || ka == nil {
					if ctx.Err() != nil {
						return
					}
					if !r.reRegister(ctx, fmt.Errorf("keepalive channel closed")) {
						if !sleepWithContext(ctx, keepAliveRetryDelay) {
							return
						}
					}
					continue outer
				}
			}
		}
	}
}

func (r *SvcDiscoveryRegistryImpl) reRegister(ctx context.Context, cause error) bool {
	r.regMu.Lock()
	defer r.regMu.Unlock()

	if r.client == nil || r.registeredService == "" || r.registeredHost == "" {
		return false
	}

	service := r.registeredService
	host := r.registeredHost
	port := r.registeredPort
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	oldLeaseID := r.leaseID

	log.ZWarn(
		context.Background(),
		"etcd keepalive lost, re-registering endpoint",
		cause,
		zap.String("service", service),
		zap.String("addr", addr),
		zap.Int64("oldLeaseID", int64(oldLeaseID)),
	)

	retryCtx, cancel := withTimeout(ctx, defaultRegisterTimeout)
	defer cancel()

	if err := r.registerLocked(retryCtx, service, host, port); err != nil {
		log.ZWarn(
			context.Background(),
			"re-register endpoint failed",
			err,
			zap.String("service", service),
			zap.String("addr", addr),
			zap.Int64("oldLeaseID", int64(oldLeaseID)),
		)

		return false
	}

	newLeaseID := r.leaseID

	if oldLeaseID != 0 && oldLeaseID != newLeaseID {
		if _, err := r.client.Revoke(context.Background(), oldLeaseID); err != nil {
			log.ZWarn(
				context.Background(),
				"failed to revoke old lease after re-register",
				err,
				zap.String("service", service),
				zap.String("addr", addr),
				zap.Int64("oldLeaseID", int64(oldLeaseID)),
				zap.Int64("newLeaseID", int64(newLeaseID)),
			)
		}
	}

	return true
}

// UnRegister removes the service endpoint from etcd
func (r *SvcDiscoveryRegistryImpl) UnRegister() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCloseTimeout)
	defer cancel()

	r.regMu.Lock()
	if r.keepAliveCancel != nil {
		r.keepAliveCancel()
		r.keepAliveCancel = nil
	}

	mgr := r.endpointMgr
	serviceKey := r.serviceKey
	leaseID := r.leaseID
	client := r.client

	r.endpointMgr = nil
	r.serviceKey = ""
	r.leaseID = 0
	r.registeredService = ""
	r.registeredHost = ""
	r.registeredPort = 0
	r.regMu.Unlock()

	if mgr == nil || serviceKey == "" {
		return nil
	}

	if err := mgr.DeleteEndpoint(ctx, serviceKey); err != nil {
		return err
	}

	if leaseID != 0 && client != nil {
		if _, err := client.Revoke(ctx, leaseID); err != nil {
			log.ZWarn(ctx, "failed to revoke lease during unregister", err, zap.String("serviceKey", serviceKey))
		}
	}

	return nil
}

// Close closes the etcd client connection
func (r *SvcDiscoveryRegistryImpl) Close() {
	r.stopServiceWatches()
	r.stopKeyWatches()
	r.stopKVKeepAlives()

	if err := r.UnRegister(); err != nil {
		log.ZWarn(context.Background(), "failed to unregister on close", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.resetConnMap()
	r.serviceDialOptions = make(map[string][]grpc.DialOption)
	if r.client != nil {
		_ = r.client.Close()
		r.client = nil
	}
}
