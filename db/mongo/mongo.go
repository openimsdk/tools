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

package mongo

import (
	"context"
	"time"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/mw/specialerror"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoConfig represents the MongoDB configuration.
type Config struct {
	Uri         string
	Address     []string
	Database    string
	Username    string
	Password    string
	MaxPoolSize int
	MaxRetry    int
	ConnTimeout time.Duration
}

// MongoDB represents a MongoDB client.
type Client struct {
	Client *mongo.Client
	Config *Config
}

// NewMongoDB initializes a new MongoDB connection.
func NewMongoDB(ctx context.Context, config *Config) (*Client, error) {

	if err := specialerror.AddReplace(mongo.ErrNoDocuments, errs.ErrRecordNotFound); err != nil {
		return nil, err
	}

	if err := config.ValidateAndSetDefaults(); err != nil {
		return nil, err
	}

	clientOpts := options.Client().ApplyURI(config.Uri).SetMaxPoolSize(uint64(config.MaxPoolSize))

	mongoClient, err := connectWithRetry(ctx, clientOpts, config.MaxRetry, config.ConnTimeout)
	if err != nil {
		return nil, err
	}

	return &Client{Client: mongoClient, Config: config}, nil
}

// connectWithRetry attempts to connect to MongoDB with retries on failure.
func connectWithRetry(ctx context.Context, clientOpts *options.ClientOptions, maxRetry int, connTimeout time.Duration) (*mongo.Client, error) {
	var mongoClient *mongo.Client
	var err error

	for attempt := 0; attempt <= maxRetry; attempt++ {
		ctx, cancel := context.WithTimeout(ctx, connTimeout)
		defer cancel()

		mongoClient, err = mongo.Connect(ctx, clientOpts)
		if err == nil && mongoClient.Ping(ctx, nil) == nil {
			return mongoClient, nil
		}

		if !shouldRetry(err) || attempt == maxRetry {
			break
		}

		time.Sleep(time.Second * time.Duration(attempt+1))
	}

	return nil, errs.WrapMsg(err, "failed to connect to MongoDB", "URI", clientOpts.GetURI())
}
