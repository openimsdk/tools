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
	"context"

	"github.com/openimsdk/tools/errs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CheckMongo tests the MongoDB connection without retries.
func CheckMongo(ctx context.Context, config *Config) error {
	if err := config.ValidateAndSetDefaults(); err != nil {
		return err
	}

	clientOpts := options.Client().ApplyURI(config.Uri)
	mongoClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return errs.WrapMsg(err, "MongoDB connect failed", "URI", config.Uri, "Database", config.Database, "MaxPoolSize", config.MaxPoolSize)
	}
	defer mongoClient.Disconnect(ctx)

	if err = mongoClient.Ping(ctx, nil); err != nil {
		return errs.WrapMsg(err, "MongoDB ping failed", "URI", config.Uri, "Database", config.Database, "MaxPoolSize", config.MaxPoolSize)
	}

	return nil
}

// ValidateAndSetDefaults validates the configuration and sets default values.
func (c *Config) ValidateAndSetDefaults() error {
	if c.Uri == "" && len(c.Address) == 0 {
		return errs.New("either Uri or Address must be provided")
	}
	if c.Database == "" {
		return errs.New("database is required")
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
