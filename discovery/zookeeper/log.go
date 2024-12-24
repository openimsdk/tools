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

package zookeeper

import (
	"context"

	"github.com/openimsdk/tools/log"
)

type nilLog struct{}

func (nilLog) Debug(ctx context.Context, msg string, keysAndValues ...any) {}

func (nilLog) Info(ctx context.Context, msg string, keysAndValues ...any) {}

func (nilLog) Warn(ctx context.Context, msg string, err error, keysAndValues ...any) {}

func (nilLog) Error(ctx context.Context, msg string, err error, keysAndValues ...any) {}

func (nilLog) Printf(string, ...interface{}) {}

func (nilLog) WithValues(keysAndValues ...any) log.Logger {
	return nilLog{}
}

func (nilLog) WithName(name string) log.Logger {
	return nilLog{}
}

func (nilLog) WithCallDepth(depth int) log.Logger {
	return nilLog{}
}

func (nilLog) Panic(ctx context.Context, msg string, err error, keysAndValues ...any) {}
