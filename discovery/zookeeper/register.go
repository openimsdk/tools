package zookeeper

import (
	"context"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/openimsdk/tools/errs"
	"google.golang.org/grpc"
)

func (s *ZkClient) CreateRpcRootNodes(serviceNames []string) error {
	for _, serviceName := range serviceNames {
		if err := s.ensureName(serviceName); err != nil && err != zk.ErrNodeExists {
			return err
		}
	}
	return nil
}

func (s *ZkClient) CreateTempNode(rpcRegisterName, addr string) (node string, err error) {
	node, err = s.conn.CreateProtectedEphemeralSequential(
		s.getPath(rpcRegisterName)+"/"+addr+"_",
		[]byte(addr),
		zk.WorldACL(zk.PermAll),
	)
	if err != nil {
		return "", errs.WrapMsg(err, "CreateProtectedEphemeralSequential failed", "path", s.getPath(rpcRegisterName)+"/"+addr+"_")
	}
	return node, nil
}

func (s *ZkClient) Register(ctx context.Context, rpcRegisterName, host string, port int, opts ...grpc.DialOption) error {
	if err := s.ensureName(rpcRegisterName); err != nil {
		return err
	}
	addr := s.getAddr(host, port)
	_, err := grpc.Dial(addr, opts...)
	if err != nil {
		return errs.WrapMsg(err, "grpc dial error", "addr", addr)
	}
	node, err := s.CreateTempNode(rpcRegisterName, addr)
	if err != nil {
		return err
	}
	s.rpcRegisterName = rpcRegisterName
	s.rpcRegisterAddr = addr
	s.node = node
	s.isRegistered = true
	return nil
}

func (s *ZkClient) UnRegister() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	err := s.conn.Delete(s.node, -1)
	if err != nil {
		return errs.WrapMsg(err, "delete node error", "node", s.node)
	}
	time.Sleep(time.Second)
	s.node = ""
	s.rpcRegisterName = ""
	s.rpcRegisterAddr = ""
	s.isRegistered = false
	s.localConns = make(map[string][]grpc.ClientConnInterface)
	s.resolvers = make(map[string]*Resolver)
	return nil
}
