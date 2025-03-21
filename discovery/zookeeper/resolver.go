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
	"strings"

	"google.golang.org/grpc/resolver"
)

type Resolver struct {
	client         *ZkClient
	target         resolver.Target
	cc             resolver.ClientConn
	addrs          []resolver.Address
	getConnsRemote func(ctx context.Context, serviceName string) (conns []resolver.Address, err error)
}

func (r *Resolver) ResolveNowZK(o resolver.ResolveNowOptions) {
	serviceName := strings.TrimLeft(r.target.URL.Path, "/")
	r.client.logger.Debug(context.Background(), "start resolve now", "target", r.target, "serviceName", serviceName)
	newConns, err := r.getConnsRemote(context.Background(), serviceName)
	if err != nil {
		r.client.logger.Error(context.Background(), "resolve now error", err, "target", r.target, "serviceName", serviceName)
		return
	}
	r.addrs = newConns
	if err := r.cc.UpdateState(resolver.State{Addresses: newConns}); err != nil {
		r.client.logger.Error(context.Background(), "UpdateState error, conns is nil from svr", err, "conns", newConns, "zk path", r.target.URL.Path, "serviceName", serviceName)
		return
	}
	r.client.logger.Debug(context.Background(), "resolve now finished", "target", r.target, "conns", r.addrs, "serviceName", serviceName)
}

func (r *Resolver) ResolveNow(o resolver.ResolveNowOptions) {}

func (r *Resolver) Close() {}

func (s *ZkClient) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	s.logger.Debug(context.Background(), "build resolver", "target", target, "cc", cc.UpdateState)
	serviceName := strings.TrimLeft(target.URL.Path, "/")
	r := &Resolver{client: s}
	r.target = target
	r.cc = cc
	r.getConnsRemote = s.GetConnsRemote
	r.ResolveNowZK(resolver.ResolveNowOptions{})
	s.lock.Lock()
	defer s.lock.Unlock()
	s.resolvers[serviceName] = r
	s.logger.Debug(context.Background(), "build resolver finished", "target", target, "cc", cc.UpdateState, "key", serviceName)
	return r, nil
}

func (s *ZkClient) Scheme() string { return s.scheme }
