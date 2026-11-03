// Copyright © 2024 OpenIM open source community. All rights reserved.
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

package mongoutil

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

const (
	defaultMaxPoolSize = 100
	defaultMaxRetry    = 3

	mongoDriverCodeUnauthorized         = 13
	mongoDriverCodeAuthenticationFailed = 18
)

func buildMongoURI(config *Config, authSource string) string {
	credentials := ""

	if config.Username != "" && config.Password != "" {
		credentials = fmt.Sprintf("%s:%s", config.Username, config.Password)
	}

	return fmt.Sprintf(
		"mongodb://%s@%s/%s?authSource=%s&maxPoolSize=%d",
		credentials,
		strings.Join(config.Address, ","),
		config.Database,
		authSource,
		config.MaxPoolSize,
	)
}

// shouldRetry determines whether an error should trigger a retry.
func shouldRetry(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && (cmdErr.Code == mongoDriverCodeUnauthorized || cmdErr.Code == mongoDriverCodeAuthenticationFailed) {
		return false
	}
	var drvErr driver.Error
	if errors.As(err, &drvErr) && (drvErr.Code == mongoDriverCodeUnauthorized || drvErr.Code == mongoDriverCodeAuthenticationFailed) {
		return false
	}
	var authErr *auth.Error
	if errors.As(err, &authErr) && !errors.As(authErr, new(net.Error)) {
		return false
	}
	var sse topology.ServerSelectionError
	if errors.As(err, &sse) {
		for _, server := range sse.Desc.Servers {
			if server.LastError != nil && !shouldRetry(ctx, server.LastError) {
				return false
			}
		}
	}
	return true
}

func retryDelay(attempt int) time.Duration {
	const base = 500 * time.Millisecond
	const maxDelay = 5 * time.Second
	delay := min(base<<min(attempt, 4), maxDelay)
	return delay/2 + rand.N(delay/2)
}
