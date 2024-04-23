// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zookeeper

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-zookeeper/zk"
	"github.com/openimsdk/tools/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

var (
	ErrConnIsNil               = errs.New("conn is nil")
	ErrConnIsNilButLocalNotNil = errs.New("conn is nil, but local is not nil")
)

func (s *ZkClient) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info(ctx, "zk watch ctx done")
			return
		case event := <-s.eventChan:
			s.logger.Debug(ctx, "zk eventChan recv new event", "event", event)
			switch event.Type {
			case zk.EventSession:
				switch event.State {
				case zk.StateHasSession:
					if s.isRegistered && !s.isStateDisconnected {
						s.logger.Debug(ctx, "zk session event stateHasSession, client prepare to create new temp node", "event", event)
						node, err := s.CreateTempNode(s.rpcRegisterName, s.rpcRegisterAddr)
						if err != nil {
							s.logger.Error(ctx, "zk session event stateHasSession, create temp node error", err, "event", event)
						} else {
							s.node = node
						}
					}
				case zk.StateDisconnected:
					s.isStateDisconnected = true
				case zk.StateConnected:
					s.isStateDisconnected = false
				default:
					s.logger.Debug(ctx, "zk session event", "event", event)
				}
			case zk.EventNodeChildrenChanged:
				s.logger.Debug(ctx, "zk event", "event", event)
				l := strings.Split(event.Path, "/")
				if len(l) > 1 {
					serviceName := l[len(l)-1]
					s.lock.Lock()
					s.flushResolverAndDeleteLocal(serviceName)
					s.lock.Unlock()
				}
				s.logger.Debug(ctx, "zk event handle success", "path", event.Path)
			case zk.EventNodeDataChanged:
			case zk.EventNodeCreated:
				s.logger.Debug(ctx, "zk node create event", "event", event)
			case zk.EventNodeDeleted:
			case zk.EventNotWatching:
			}
		}
	}
}

func (s *ZkClient) GetConnsRemote(ctx context.Context, serviceName string) (conns []resolver.Address, err error) {
	err = s.ensureName(serviceName)
	if err != nil {
		return nil, err
	}

	path := s.getPath(serviceName)
	_, _, _, err = s.conn.ChildrenW(path)
	if err != nil {
		return nil, errs.WrapMsg(err, "children watch error", "path", path)
	}
	childNodes, _, err := s.conn.Children(path)
	if err != nil {
		return nil, errs.WrapMsg(err, "get children error", "path", path)
	} else {
		for _, child := range childNodes {
			fullPath := path + "/" + child
			data, _, err := s.conn.Get(fullPath)
			if err != nil {
				return nil, errs.WrapMsg(err, "get children error", "fullPath", fullPath)
			}
			s.logger.Debug(ctx, "get addr from remote", "conn", string(data))
			conns = append(conns, resolver.Address{Addr: string(data), ServerName: serviceName})
		}
	}
	return conns, nil
}

func (s *ZkClient) GetUserIdHashGatewayHost(ctx context.Context, userId string) (string, error) {
	s.logger.Warn(ctx, "not implement", errs.New("zkclinet not implement GetUserIdHashGatewayHost method"))
	return "", nil
}

func (s *ZkClient) GetConns(ctx context.Context, serviceName string, opts ...grpc.DialOption) ([]*grpc.ClientConn, error) {
	s.logger.Debug(ctx, "get conns from client", "serviceName", serviceName)
	s.lock.Lock()
	defer s.lock.Unlock()
	conns := s.localConns[serviceName]
	if len(conns) == 0 {
		s.logger.Debug(ctx, "get conns from zk remote", "serviceName", serviceName)
		addrs, err := s.GetConnsRemote(ctx, serviceName)
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, errs.New("addr is empty").WrapMsg("no conn for service", "serviceName",
				serviceName, "local conn", s.localConns, "ZkServers", s.ZkServers, "zkRoot", s.zkRoot)
		}
		for _, addr := range addrs {
			cc, err := grpc.DialContext(ctx, addr.Addr, append(s.options, opts...)...)
			if err != nil {
				s.logger.Error(context.Background(), "dialContext failed", err, "addr", addr.Addr, "opts", append(s.options, opts...))
				return nil, errs.WrapMsg(err, "DialContext failed", "addr.Addr", addr.Addr)
			}
			conns = append(conns, cc)
		}
		s.localConns[serviceName] = conns
	}
	return conns, nil
}

func (s *ZkClient) GetConn(ctx context.Context, serviceName string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	newOpts := append(s.options, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, s.balancerName)))
	s.logger.Debug(context.Background(), "get conn from client", "serviceName", serviceName)
	return grpc.DialContext(ctx, fmt.Sprintf("%s:///%s", s.scheme, serviceName), append(newOpts, opts...)...)
}

func (s *ZkClient) GetSelfConnTarget() string {
	return s.rpcRegisterAddr
}

func (s *ZkClient) CloseConn(conn *grpc.ClientConn) {
	conn.Close()
}
