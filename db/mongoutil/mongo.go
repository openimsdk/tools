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
	"time"

	"github.com/amazing-socrates/next-tools/db/tx"
	"github.com/amazing-socrates/next-tools/errs"
	"github.com/amazing-socrates/next-tools/mw/specialerror"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// ReadPreferenceMode represents the read preference mode.
type ReadPreferenceMode string

const (
	// ReadPreferencePrimary Strongly coherent reads (reads from the master node) values.
	ReadPreferencePrimary ReadPreferenceMode = "primary"
	// ReadPreferencePrimaryPreferred Prioritize master nodes, cut slave nodes when master is unavailable
	ReadPreferencePrimaryPreferred ReadPreferenceMode = "primaryPreferred"
	// ReadPreferenceSecondary Reads from slave nodes
	ReadPreferenceSecondary ReadPreferenceMode = "secondary"
	// ReadPreferenceSecondaryPreferred Extended read capability (prioritize reads from nodes)
	ReadPreferenceSecondaryPreferred ReadPreferenceMode = "secondaryPreferred"
	// ReadPreferenceNearest Low latency reads (nearby nodes)
	ReadPreferenceNearest ReadPreferenceMode = "nearest"
)

func init() {
	if err := specialerror.AddReplace(mongo.ErrNoDocuments, errs.ErrRecordNotFound); err != nil {
		panic(err)
	}
}

// Config represents the MongoDB configuration.
type Config struct {
	Uri                         string
	Address                     []string
	Database                    string
	Username                    string
	Password                    string
	AuthSource                  string
	ReadPreference              ReadPreferenceMode
	NeedReadPrefMaxStaleness    bool
	ReadPrefMaxStaleness        time.Duration
	TLSEnabled                  bool
	TlsCAFile                   string
	TlsAllowInvalidCertificates bool
	MaxPoolSize                 int
	MinPoolSize                 int
	MaxRetry                    int
	RetryWrites                 bool
	RetryReads                  bool
}

func (c *Config) SetReadPreference(opts *options.ClientOptions) *options.ClientOptions {
	if opts == nil {
		return opts
	}
	switch c.ReadPreference {
	case ReadPreferencePrimary:
		readPref := readpref.Primary()
		opts.SetReadPreference(readPref)
	case ReadPreferencePrimaryPreferred:
		readPref := readpref.PrimaryPreferred()
		if c.NeedReadPrefMaxStaleness {
			readpref.PrimaryPreferred(
				readpref.WithMaxStaleness(c.ReadPrefMaxStaleness),
			)
		}
		opts.SetReadPreference(readPref)
	case ReadPreferenceSecondary:
		readPref := readpref.Secondary()
		if c.NeedReadPrefMaxStaleness {
			readpref.Secondary(
				readpref.WithMaxStaleness(c.ReadPrefMaxStaleness),
			)
		}
		opts.SetReadPreference(readPref)
	case ReadPreferenceSecondaryPreferred:
		readPref := readpref.SecondaryPreferred()
		if c.NeedReadPrefMaxStaleness {
			readpref.SecondaryPreferred(
				readpref.WithMaxStaleness(c.ReadPrefMaxStaleness),
			)
		}
		opts.SetReadPreference(readPref)
	case ReadPreferenceNearest:
		readPref := readpref.Nearest()
		if c.NeedReadPrefMaxStaleness {
			readpref.Nearest(
				readpref.WithMaxStaleness(c.ReadPrefMaxStaleness),
			)
		}
		opts.SetReadPreference(readPref)
	default:
		readPref := readpref.Primary()
		opts.SetReadPreference(readPref)
	}
	return opts
}

type Client struct {
	tx tx.Tx
	db *mongo.Database
}

func (c *Client) GetDB() *mongo.Database {
	return c.db
}

func (c *Client) GetTx() tx.Tx {
	return c.tx
}

// NewMongoDB initializes a new MongoDB connection.
func NewMongoDB(ctx context.Context, config *Config) (*Client, error) {
	if err := config.ValidateAndSetDefaults(); err != nil {
		return nil, err
	}

	opts := options.Client().ApplyURI(config.Uri).
		SetMaxPoolSize(uint64(config.MaxPoolSize)).
		SetMinPoolSize(uint64(config.MinPoolSize)).
		SetRetryWrites(config.RetryWrites).
		SetRetryReads(config.RetryReads)

	opts = config.SetReadPreference(opts)

	var (
		cli *mongo.Client
		err error
	)
	for i := 0; i < config.MaxRetry; i++ {
		cli, err = connectMongo(ctx, opts)
		if err != nil && shouldRetry(ctx, err) {
			time.Sleep(time.Second / 2)
			continue
		}
		break
	}
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to connect to MongoDB", "URI", config.Uri)
	}
	mtx, err := NewMongoTx(ctx, cli)
	if err != nil {
		return nil, err
	}
	return &Client{
		tx: mtx,
		db: cli.Database(config.Database),
	}, nil
}

func connectMongo(ctx context.Context, opts *options.ClientOptions) (*mongo.Client, error) {
	cli, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := cli.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return cli, nil
}
