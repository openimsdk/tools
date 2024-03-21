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
type MongoConfig struct {
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
type MongoDB struct {
	Client *mongo.Client
	Config *MongoConfig
}

// NewMongoDB initializes a new MongoDB connection.
func NewMongoDB(ctx context.Context, config *MongoConfig) (*MongoDB, error) {
	specialerror.AddReplace(mongo.ErrNoDocuments, errs.ErrRecordNotFound)

	if err := config.ValidateAndSetDefaults(); err != nil {
		return nil, err
	}

	clientOpts := options.Client().ApplyURI(config.Uri).SetMaxPoolSize(uint64(config.MaxPoolSize))

	mongoClient, err := connectWithRetry(ctx, clientOpts, config.MaxRetry, config.ConnTimeout)
	if err != nil {
		return nil, err
	}

	return &MongoDB{Client: mongoClient, Config: config}, nil
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

// CheckMongo tests the MongoDB connection without retries.
func CheckMongo(ctx context.Context, config *MongoConfig) error {

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
