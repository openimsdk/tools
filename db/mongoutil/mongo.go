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

	"github.com/openimsdk/tools/db/tx"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mw/specialerror"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/tag"
)

// MongoMode
const (
	StandaloneMode = "standalone" // Standalone MongoDB mode
	ReplicaSetMode = "replicaSet" // Replica set MongoDB mode
)

// ReadPreference
const (
	PrimaryMode            = "primary"            // Primary read preference mode
	PrimaryPreferredMode   = "primaryPreferred"   // Primary preferred read preference mode
	SecondaryMode          = "secondary"          // Secondary read preference mode
	SecondaryPreferredMode = "secondaryPreferred" // Secondary preferred read preference mode
	NearestMode            = "nearest"            // Nearest read preference mode
)

// WriteConcern
const (
	MajorityWriteConcern = "majority" // Majority write concern level
	// JournalWriteConcern  = "journal"  // Journal write concern level
)

// ReadConcern levels
const (
	LocalReadConcern        = "local"        // Local read concern level
	AvailableReadConcern    = "available"    // Available read concern level
	MajorityReadConcern     = "majority"     // Majority read concern level
	LinearizableReadConcern = "linearizable" // Linearizable read concern level
	SnapshotReadConcern     = "snapshot"     // Snapshot read concern level
)

func init() {
	if err := specialerror.AddReplace(mongo.ErrNoDocuments, errs.ErrRecordNotFound); err != nil {
		panic(err)
	}
}

// Config represents the MongoDB configuration.
type Config struct {
	Uri         string
	Address     []string
	Database    string
	Username    string
	Password    string
	AuthSource  string
	MaxPoolSize int
	MaxRetry    int
	MongoMode   string // "replicaSet" or "standalone"

	ReplicaSet     *ReplicaSetConfig
	ReadPreference *ReadPrefConfig
	WriteConcern   *WriteConcernConfig
}

type ReplicaSetConfig struct {
	Name         string        `json:"name" yaml:"name" config:"allowempty"`                 // Replica set name
	Hosts        []string      `json:"hosts" yaml:"hosts" config:"allowempty"`               // Replica set host list
	ReadConcern  string        `json:"readConcern" yaml:"readConcern" config:"allowempty"`   // Read concern level: local, available, majority, linearizable, snapshot
	MaxStaleness time.Duration `json:"maxStaleness" yaml:"maxStaleness" config:"allowempty"` // Maximum staleness time
}

type ReadPrefConfig struct {
	Mode         string              `json:"mode" yaml:"mode" config:"allowempty"`                 // primary, secondary, secondaryPreferred, nearest
	TagSets      []map[string]string `json:"tagSets" yaml:"tagSets" config:"allowempty"`           // Tag sets
	MaxStaleness time.Duration       `json:"maxStaleness" yaml:"maxStaleness" config:"allowempty"` // Maximum staleness time
}

type WriteConcernConfig struct {
	W        any           `json:"w" yaml:"w" config:"allowempty"`               // Write node count or tag (int, "majority", or custom tag)
	J        bool          `json:"j" yaml:"j" config:"allowempty"`               // Whether to wait for journal confirmation
	WTimeout time.Duration `json:"wtimeout" yaml:"wtimeout" config:"allowempty"` // Write timeout duration
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

	var opts *options.ClientOptions

	if config.MongoMode == ReplicaSetMode {
		if config.ReplicaSet == nil {
			return nil, errs.New("replicaSet configuration required for replicaSet mode")
		}
		if config.ReplicaSet.Name == "" {
			return nil, errs.New("replicaSet name required for replicaSet mode")
		}
		if len(config.ReplicaSet.Hosts) == 0 && len(config.Address) == 0 {
			return nil, errs.New("replicaSet hosts or address required for replicaSet mode")
		}
	} else {
		if config.Uri == "" && config.Address == nil {
			return nil, errs.New("address required for standalone mode")
		}
	}

	switch config.MongoMode {
	case ReplicaSetMode:
		opts = buildReplicaSetOptions(config)
	case StandaloneMode:
		if err := config.ValidateAndSetDefaults(); err != nil {
			return nil, err
		}

		opts = options.Client().ApplyURI(config.Uri).SetMaxPoolSize(uint64(config.MaxPoolSize))
	}

	var (
		cli *mongo.Client
		err error
	)

	for range config.MaxRetry {
		cli, err = connectMongo(ctx, opts)
		if err != nil && shouldRetry(ctx, err) {
			log.ZError(ctx, "Fail to connect Mongo", err)
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

func buildReplicaSetOptions(config *Config) *options.ClientOptions {
	opts := options.Client()

	hosts := config.ReplicaSet.Hosts
	if len(hosts) == 0 {
		hosts = config.Address
	}
	opts.SetHosts(hosts)

	opts.SetReplicaSet(config.ReplicaSet.Name)

	if config.Username != "" && config.Password != "" {
		credential := options.Credential{
			Username:   config.Username,
			Password:   config.Password,
			AuthSource: config.AuthSource,
		}
		if credential.AuthSource == "" {
			credential.AuthSource = config.Database
		}
		opts.SetAuth(credential)
	}

	if config.MaxPoolSize > 0 {
		opts.SetMaxPoolSize(uint64(config.MaxPoolSize))
	}

	if config.ReadPreference != nil {
		readPref := buildReadPreference(config.ReadPreference)
		opts.SetReadPreference(readPref)
	}

	if config.WriteConcern != nil {
		writeConcern := buildWriteConcern(config.WriteConcern)
		opts.SetWriteConcern(writeConcern)
	}

	if config.ReplicaSet.ReadConcern != "" {
		readConcern := buildReadConcern(config.ReplicaSet.ReadConcern)
		opts.SetReadConcern(readConcern)
	}

	return opts
}

func buildReadPreference(config *ReadPrefConfig) *readpref.ReadPref {
	var mode readpref.Mode
	switch config.Mode {
	case PrimaryMode:
		mode = readpref.PrimaryMode
	case PrimaryPreferredMode:
		mode = readpref.PrimaryPreferredMode
	case SecondaryMode:
		mode = readpref.SecondaryMode
	case SecondaryPreferredMode:
		mode = readpref.SecondaryPreferredMode
	case NearestMode:
		mode = readpref.NearestMode
	default:
		mode = readpref.PrimaryMode
	}

	opts := make([]readpref.Option, 0)

	if len(config.TagSets) > 0 {
		tagSets := tag.NewTagSetsFromMaps(config.TagSets)
		opts = append(opts, readpref.WithTagSets(tagSets...))
	}

	if config.MaxStaleness > 0 {
		opts = append(opts, readpref.WithMaxStaleness(config.MaxStaleness))
	}

	readPref, _ := readpref.New(mode, opts...)
	return readPref
}

func buildWriteConcern(config *WriteConcernConfig) *writeconcern.WriteConcern {
	wc := &writeconcern.WriteConcern{}

	switch w := config.W.(type) {
	case int:
		wc.W = w
	case string:
		if w == MajorityWriteConcern { // Use majority
			wc.W = MajorityWriteConcern
		} else { // Use custom tag
			wc.W = w
		}
	}

	if config.J {
		wc.Journal = &config.J
	}

	if config.WTimeout > 0 {
		wc.WTimeout = config.WTimeout
	}

	return wc
}

func buildReadConcern(level string) *readconcern.ReadConcern {
	switch level {
	case LocalReadConcern:
		return readconcern.Local()
	case AvailableReadConcern:
		return readconcern.Available()
	case MajorityReadConcern:
		return readconcern.Majority()
	case LinearizableReadConcern:
		return readconcern.Linearizable()
	case SnapshotReadConcern:
		return readconcern.Snapshot()
	default:
		return readconcern.Local()
	}
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
