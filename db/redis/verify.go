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

package cache

import (
	"context"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/utils"
	"github.com/redis/go-redis/v9"
)

// CheckRedis checks the Redis connection.
func CheckRedis(ctx context.Context, config *RedisConfig) error {

	redisInfo, err := utils.JsonMarshal(config)
	if err != nil {
		return errs.WrapMsg(err, "Failed to marshal Redis config.")
	}

	client, err := NewRedisClient(ctx, config)
	if err != nil {
		return errs.WrapMsg(err, "Failed to initialize Redis client.", "Config", string(redisInfo))
	}

	defer client.(*redis.Client).Close()

	// Ping the Redis server to check connectivity.
	if err := client.Ping(ctx).Err(); err != nil {
		return errs.WrapMsg(err, "Redis ping failed.", "Config", string(redisInfo))
	}

	return nil
}
