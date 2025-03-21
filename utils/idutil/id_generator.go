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

package idutil

import (
	"github.com/openimsdk/tools/utils/encrypt"
	"github.com/openimsdk/tools/utils/stringutil"
	"github.com/openimsdk/tools/utils/timeutil"
	"math/rand"
	"strconv"
	"time"
)

func GetMsgIDByMD5(sendID string) string {
	t := stringutil.Int64ToString(timeutil.GetCurrentTimestampByNano())
	return encrypt.Md5(t + sendID + stringutil.Int64ToString(rand.Int63n(timeutil.GetCurrentTimestampByNano())))
}

func OperationIDGenerator() string {
	return strconv.FormatInt(time.Now().UnixNano()+int64(rand.Uint32()), 10)
}
