package etcd

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/openimsdk/tools/log"
)

type registrar struct {
	client        *clientv3.Client
	rootDirectory string

	mu              sync.Mutex
	endpointMgr     endpoints.Manager
	serviceKey      string           // e.g. /openim/msg/192.168.1.10:10001
	leaseID         clientv3.LeaseID // etcd current lease ID
	target          string           // self exposed host:port (for IsSelfNode)
	keepAliveCancel context.CancelFunc

	// on reconnecting
	service string
	host    string
	port    int
}

func newRegistrar(client *clientv3.Client, rootDirectory string) *registrar {
	return &registrar{
		client:        client,
		rootDirectory: rootDirectory,
	}
}

// Register registers a new service endpoint with etcd
func (r *registrar) Register(ctx context.Context, serviceName, host string, port int, opts ...grpc.DialOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()

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

func (r *registrar) registerLocked(ctx context.Context, serviceName, host string, port int) error {
	if ctx == nil {
		ctx = context.Background()
	}

	serviceDir := fmt.Sprintf("%s/%s", r.rootDirectory, serviceName)
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
	r.target = endpointAddr
	r.service = serviceName
	r.host = host
	r.port = port

	log.ZInfo(ctx, "rpc register success",
		"service", serviceName,
		"serviceKey", serviceKey,
		"target", endpointAddr,
		"leaseID", int64(leaseResp.ID),
	)

	return nil
}

func (r *registrar) keepAliveLoop(ctx context.Context) {
outer:
	for {
		if ctx.Err() != nil {
			return
		}
		client := r.client
		if client == nil {
			return
		}

		r.mu.Lock()
		leaseID := r.leaseID
		r.mu.Unlock()
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

func (r *registrar) reRegister(ctx context.Context, cause error) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.client == nil || r.service == "" || r.host == "" {
		return false
	}

	service := r.service
	host := r.host
	port := r.port
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
func (r *registrar) UnRegister() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCloseTimeout)
	defer cancel()

	r.mu.Lock()
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
	r.target = ""
	r.service = ""
	r.host = ""
	r.port = 0
	r.mu.Unlock()

	if mgr == nil || serviceKey == "" {
		return nil
	}

	if err := mgr.DeleteEndpoint(ctx, serviceKey); err != nil {
		log.ZWarn(ctx, "failed to delete endpoint during unregister", err, "serviceKey", serviceKey)
		return err
	}

	log.ZInfo(ctx, "rpc unregister success", "serviceKey", serviceKey)

	if leaseID != 0 && client != nil {
		if _, err := client.Revoke(ctx, leaseID); err != nil {
			log.ZWarn(ctx, "failed to revoke lease during unregister", err, "serviceKey", serviceKey)
		}
	}

	return nil
}

// GetSelfConnTarget returns the connection target for the current service
func (r *registrar) GetSelfConnTarget() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.target
}

// IsSelfNode checks if the given client conn connects to the current service
func (r *registrar) IsSelfNode(cc grpc.ClientConnInterface) bool {
	cli, ok := cc.(*grpc.ClientConn)
	if !ok {
		return false
	}
	target := r.GetSelfConnTarget()
	return target != "" && target == cli.Target()
}

func (r *registrar) close() error {
	return r.UnRegister()
}
