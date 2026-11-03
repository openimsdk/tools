package mongoutil

import (
	"context"

	"github.com/openimsdk/tools/errs"
)

// Check tests the MongoDB connection without retries.
func Check(ctx context.Context, config *Config) error {
	client, err := NewMongoDB(ctx, config)
	if err != nil {
		if config.MongoMode == ReplicaSetMode {
			return errs.WrapMsg(err, "failed to connect to MongoDB replica set", "config", config)
		}

		return errs.WrapMsg(err, "MongoDB connect failed", "URI", config.Uri, "Database", config.Database, "MaxPoolSize", config.MaxPoolSize)
	}

	defer func() {
		if client != nil && client.db != nil {
			_ = client.db.Client().Disconnect(ctx)
		}
	}()

	return nil
}

// ValidateAndSetDefaults validates the configuration and sets default values.
func (c *Config) ValidateAndSetDefaults() error {
	if c.MongoMode == "" {
		c.MongoMode = StandaloneMode
	}
	if c.Database == "" {
		return errs.Wrap(errs.New("database is required"))
	}
	if c.MaxPoolSize <= 0 {
		c.MaxPoolSize = defaultMaxPoolSize
	}
	if c.MaxRetry <= 0 {
		c.MaxRetry = defaultMaxRetry
	}
	switch c.MongoMode {
	case ReplicaSetMode:
		if c.ReplicaSet == nil {
			return errs.Wrap(errs.New("replicaSet configuration required for replicaSet mode"))
		}
		if c.ReplicaSet.Name == "" {
			return errs.Wrap(errs.New("replicaSet name required for replicaSet mode"))
		}
		if len(c.ReplicaSet.Hosts) == 0 && len(c.Address) == 0 {
			return errs.Wrap(errs.New("replicaSet hosts or address required for replicaSet mode"))
		}
	case StandaloneMode:
		if c.Uri == "" && len(c.Address) == 0 {
			return errs.Wrap(errs.New("either Uri or Address must be provided"))
		}
		if c.Uri == "" {
			// if authSource is not provided, default to database name
			if c.AuthSource == "" {
				c.Uri = buildMongoURI(c, c.Database)
			} else {
				c.Uri = buildMongoURI(c, c.AuthSource)
			}
		}
	default:
		return errs.Wrap(errs.New("unsupported mongo mode", "mode", c.MongoMode))
	}
	return nil
}
