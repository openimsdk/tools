package unrelation

import (
	"context"
	"time"

	"github.com/openimsdk/tools/log"

	"github.com/openimsdk/tools/pkg/common/config"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/mw/specialerror"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoDB Initialize MongoDB connection.
func NewMongoDB(ctx context.Context, mongoConf *config.Mongo) (*Mongo, error) {
	specialerror.AddReplace(mongo.ErrNoDocuments, errs.ErrRecordNotFound)
	uri := buildMongoURI(mongoConf)

	var mongoClient *mongo.Client
	var err error

	// Retry connecting to MongoDB
	for i := 0; i <= maxRetry; i++ {
		ctx, cancel := context.WithTimeout(ctx, mongoConnTimeout)
		defer cancel()
		mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err == nil {
			if err = mongoClient.Ping(ctx, nil); err != nil {
				return nil, errs.WrapMsg(err, uri)
			}
			log.CInfo(ctx, "MONGODB connected successfully", "uri", uri)
			return &Mongo{db: mongoClient, mongoConf: mongoConf}, nil
		}
		if shouldRetry(err) {
			time.Sleep(time.Second) // exponential backoff could be implemented here
			continue
		}
	}
	return nil, errs.WrapMsg(err, uri)
}
