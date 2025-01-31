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
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

const (
	defaultMaxPoolSize = 100
	defaultMaxRetry    = 3
)

func buildMongoURI(config *Config, authSource string) string {
	credentials := ""

	if config.Username != "" && config.Password != "" {
		credentials = fmt.Sprintf("%s:%s", config.Username, config.Password)
	}

	uri := fmt.Sprintf(
		"mongodb://%s@%s/%s?authSource=%s&maxPoolSize=%d",
		credentials,
		strings.Join(config.Address, ","),
		config.Database,
		authSource,
		config.MaxPoolSize,
	)

	values := url.Values{}
	values.Add("tls", strconv.FormatBool(config.TLSEnabled))

	if config.TLSEnabled {
		if config.TlsCAFile != "" {
			values.Add("tlsCAFile", config.TlsCAFile)
		}
		if config.TlsAllowInvalidCertificates {
			values.Add("tlsAllowInvalidCertificates", strconv.FormatBool(config.TlsAllowInvalidCertificates))
		}
	}

	tlsConfig := values.Encode()
	if tlsConfig != "" {
		uri += "&" + tlsConfig
	}

	return uri
}

// shouldRetry determines whether an error should trigger a retry.
func shouldRetry(ctx context.Context, err error) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		if cmdErr, ok := err.(mongo.CommandError); ok {
			return cmdErr.Code != 13 && cmdErr.Code != 18
		}
		return true
	}
}
