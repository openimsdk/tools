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

package mongo

import (
	"errors"

	"github.com/openimsdk/tools/errs"
)

// ValidateAndSetDefaults validates the configuration and sets default values.
func (c *MongoConfig) ValidateAndSetDefaults() error {
	if c.Uri == "" && len(c.Address) == 0 {
		return errs.Wrap(errors.New("either Uri or Address must be provided"))
	}
	if c.Database == "" {
		return errs.Wrap(errors.New("database is required"))
	}
	if c.MaxPoolSize <= 0 {
		c.MaxPoolSize = DefaultMaxPoolSize
	}
	if c.MaxRetry < 0 {
		c.MaxRetry = DefaultMaxRetry
	}
	if c.ConnTimeout <= 0 {
		c.ConnTimeout = DefaultConnTimeout
	}
	if c.Uri == "" {
		c.Uri = buildMongoURI(c)
	}
	return nil
}
