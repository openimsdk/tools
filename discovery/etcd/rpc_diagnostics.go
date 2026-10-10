package etcd

import (
	"context"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"

	"github.com/openimsdk/tools/log"
)

type rpcAttemptState struct {
	mu         sync.Mutex
	method     string
	remoteAddr string
	startTime  time.Time
}

type rpcStatsCtxKey struct{}

type etcdRPCDiagnostics struct{}

func (h etcdRPCDiagnostics) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return context.WithValue(ctx, rpcStatsCtxKey{}, &rpcAttemptState{
		method:    info.FullMethodName,
		startTime: time.Now(),
	})
}

func (h etcdRPCDiagnostics) HandleRPC(ctx context.Context, s stats.RPCStats) {
	state, ok := ctx.Value(rpcStatsCtxKey{}).(*rpcAttemptState)
	if !ok || state == nil {
		return
	}

	switch stat := s.(type) {
	case *stats.OutHeader:
		if stat.RemoteAddr != nil {
			state.mu.Lock()
			state.remoteAddr = stat.RemoteAddr.String()
			state.mu.Unlock()
		}
	case *stats.End:
		state.mu.Lock()
		remoteAddr := state.remoteAddr
		method := state.method
		cost := time.Since(state.startTime)
		state.mu.Unlock()

		err := stat.Error
		if err == nil {
			log.ZDebug(ctx, "etcd rpc call finished",
				"method", method,
				"remoteAddr", remoteAddr,
				"cost", cost,
			)
			return
		}

		st, _ := status.FromError(err)
		code := st.Code()
		if code == codes.Canceled {
			log.ZDebug(ctx, "etcd rpc call canceled",
				"method", method,
				"remoteAddr", remoteAddr,
				"cost", cost,
			)
			return
		}

		log.ZWarn(ctx, "etcd rpc call failed", err,
			"method", method,
			"remoteAddr", remoteAddr,
			"code", code.String(),
			"cost", cost,
		)
	}
}

func (h etcdRPCDiagnostics) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h etcdRPCDiagnostics) HandleConn(_ context.Context, s stats.ConnStats) {
	switch cs := s.(type) {
	case *stats.ConnEnd:
		_ = cs
	}
}

var _ stats.Handler = etcdRPCDiagnostics{}

func addrString(a net.Addr) string {
	if a == nil {
		return ""
	}
	return a.String()
}
