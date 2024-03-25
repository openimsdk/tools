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
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

const (
	defaultFreq = time.Minute * 30
	timeout     = 5
)

type ZkClient struct {
	zkServers []string
	zkRoot    string
	userName  string
	password  string

	rpcRegisterName string
	rpcRegisterAddr string
	isRegistered    bool
	scheme          string

	timeout   int
	conn      *zk.Conn
	eventChan <-chan zk.Event
	node      string
	ticker    *time.Ticker

	lock    sync.Locker
	options []grpc.DialOption

	resolvers           map[string]*Resolver
	localConns          map[string][]*grpc.ClientConn
	cancel              context.CancelFunc
	isStateDisconnected bool
	balancerName        string

	logger log.Logger
}

func NewClient(zkServers []string, zkRoot string, options ...ZkOption) (*ZkClient, error) {
	client := &ZkClient{
		zkServers:  zkServers,
		zkRoot:     "/",
		scheme:     zkRoot,
		timeout:    timeout,
		localConns: make(map[string][]*grpc.ClientConn),
		resolvers:  make(map[string]*Resolver),
		lock:       &sync.Mutex{},
		logger:     nilLog{},
	}
	baseCtx, cancel := context.WithCancel(context.Background())
	client.cancel = cancel
	client.ticker = time.NewTicker(defaultFreq)
	for _, option := range options {
		option(client)
	}
	conn, eventChan, err := zk.Connect(
		zkServers,
		time.Duration(client.timeout)*time.Second,
	)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	// wait for successfully connect
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event := <-eventChan:
			if event.State == zk.StateConnected {

				goto Connected
			}
		case <-timeout:
			return nil, errs.WrapMsg(errors.New("timeout waiting for Zookeeper connection"), "Zookeeper Addr: "+strings.Join(zkServers, " "))
		}
	}

Connected:

	if client.userName != "" && client.password != "" {
		if err := conn.AddAuth("digest", []byte(client.userName+":"+client.password)); err != nil {
			return nil, errs.WrapMsg(err, "zk addAuth failed", "userName", client.userName, "password", client.password)
		}
	}

	client.zkRoot += zkRoot
	client.eventChan = eventChan
	client.conn = conn

	var errZK error
	for i := 0; i < 300; i++ {
		if errZK = client.ensureRoot(); errZK != nil {
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}
	if errZK != nil {
		return nil, errZK
	}
	resolver.Register(client)
	go client.refresh(baseCtx)
	go client.watch(baseCtx)
	time.Sleep(time.Millisecond * 50)
	return client, nil
}

func (s *ZkClient) Close() {
	s.logger.Info(context.Background(), "close zk called")
	s.cancel()
	s.ticker.Stop()
	s.conn.Close()
}

func (s *ZkClient) ensureAndCreate(node string) error {
	exists, _, err := s.conn.Exists(node)
	if err != nil {
		return errs.WrapMsg(err, "checking existence for node %s in ZkClient ensureAndCreate", "node", node)
	}
	if !exists {
		_, err = s.conn.Create(node, []byte(""), 0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return errs.WrapMsg(err, "creating node %s in ZkClient ensureAndCreate", "node", node)
		}
	}
	return nil
}

func (s *ZkClient) refresh(ctx context.Context) {
	for range s.ticker.C {
		s.logger.Debug(ctx, "zk refresh local conns")
		s.lock.Lock()
		for rpcName := range s.resolvers {
			s.flushResolver(rpcName)
		}
		for rpcName := range s.localConns {
			delete(s.localConns, rpcName)
		}
		s.lock.Unlock()
		s.logger.Debug(ctx, "zk refresh local conns success")
	}
}

func (s *ZkClient) flushResolverAndDeleteLocal(serviceName string) {
	s.logger.Debug(context.Background(), "zk start flush", "serviceName", serviceName)
	s.flushResolver(serviceName)
	delete(s.localConns, serviceName)
}

func (s *ZkClient) flushResolver(serviceName string) {
	r, ok := s.resolvers[serviceName]
	if ok {
		r.ResolveNowZK(resolver.ResolveNowOptions{})
	}
}

func (s *ZkClient) GetZkConn() *zk.Conn {
	return s.conn
}

func (s *ZkClient) GetRootPath() string {
	return s.zkRoot
}

func (s *ZkClient) GetNode() string {
	return s.node
}

func (s *ZkClient) ensureRoot() error {
	return s.ensureAndCreate(s.zkRoot)
}

func (s *ZkClient) ensureName(rpcRegisterName string) error {
	return s.ensureAndCreate(s.getPath(rpcRegisterName))
}

func (s *ZkClient) getPath(rpcRegisterName string) string {
	return s.zkRoot + "/" + rpcRegisterName
}

func (s *ZkClient) getAddr(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func (s *ZkClient) AddOption(opts ...grpc.DialOption) {
	s.options = append(s.options, opts...)
}

func (s *ZkClient) GetClientLocalConns() map[string][]*grpc.ClientConn {
	return s.localConns
}
