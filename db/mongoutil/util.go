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

	"github.com/openimsdk/tools/db/pagination"
	"github.com/openimsdk/tools/errs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func basic[T any]() bool {
	var t T
	switch any(t).(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string, []byte:
		return true
	case *int, *int8, *int16, *int32, *int64, *uint, *uint8, *uint16, *uint32, *uint64, *float32, *float64, *string, *[]byte:
		return true
	default:
		return false
	}
}

func anes[T any](ts []T) []any {
	val := make([]any, len(ts))
	for i := range ts {
		val[i] = ts[i]
	}
	return val
}

func findOptionToCountOption(opts []*options.FindOptions) *options.CountOptions {
	return options.Count()
}

func InsertMany[T any](ctx context.Context, coll *mongo.Collection, val []T, opts ...*options.InsertManyOptions) error {
	_, err := coll.InsertMany(ctx, anes(val), opts...)
	if err != nil {
		return errs.WrapMsg(err, "mongo insert many")
	}
	return nil
}

func UpdateOne(ctx context.Context, coll *mongo.Collection, filter any, update any, notMatchedErr bool, opts ...*options.UpdateOptions) error {
	res, err := coll.UpdateOne(ctx, filter, update, opts...)
	if err != nil {
		return errs.WrapMsg(err, "mongo update one")
	}
	if notMatchedErr && res.MatchedCount == 0 {
		return errs.WrapMsg(mongo.ErrNoDocuments, "mongo update not matched")
	}
	return nil
}

func UpdateOneResult(ctx context.Context, coll *mongo.Collection, filter any, update any, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	res, err := coll.UpdateOne(ctx, filter, update, opts...)
	if err != nil {
		return nil, errs.WrapMsg(err, "mongo update one")
	}
	return res, nil
}

func UpdateMany(ctx context.Context, coll *mongo.Collection, filter any, update any, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	res, err := coll.UpdateMany(ctx, filter, update, opts...)
	if err != nil {
		return nil, errs.WrapMsg(err, "mongo update many")
	}
	return res, nil
}

func Find[T any](ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.FindOptions) ([]T, error) {
	cur, err := coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, errs.WrapMsg(err, "mongo find")
	}
	defer cur.Close(ctx)
	return Decodes[T](ctx, cur)
}

func FindOne[T any](ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.FindOneOptions) (res T, err error) {
	cur := coll.FindOne(ctx, filter, opts...)
	if err := cur.Err(); err != nil {
		return res, errs.WrapMsg(err, "mongo find one")
	}
	return DecodeOne[T](cur.Decode)
}

func FindOneAndUpdate[T any](ctx context.Context, coll *mongo.Collection, filter any, update any, opts ...*options.FindOneAndUpdateOptions) (res T, err error) {
	result := coll.FindOneAndUpdate(ctx, filter, update, opts...)
	if err := result.Err(); err != nil {
		return res, errs.WrapMsg(err, "mongo find one and update")
	}
	return DecodeOne[T](result.Decode)
}

func FindPage[T any](ctx context.Context, coll *mongo.Collection, filter any, pagination pagination.Pagination, opts ...*options.FindOptions) (int64, []T, error) {
	count, err := Count(ctx, coll, filter, findOptionToCountOption(opts))
	if err != nil {
		return 0, nil, errs.WrapMsg(err, "mongo failed to count documents in collection")
	}
	if count == 0 || pagination == nil {
		return count, nil, nil
	}
	skip := int64(pagination.GetPageNumber()-1) * int64(pagination.GetShowNumber())
	if skip < 0 || skip >= count || pagination.GetShowNumber() <= 0 {
		return count, nil, nil
	}
	opt := options.Find().SetSkip(skip).SetLimit(int64(pagination.GetShowNumber()))
	res, err := Find[T](ctx, coll, filter, append(opts, opt)...)
	if err != nil {
		return 0, nil, err
	}
	return count, res, nil
}

func FindPageOnly[T any](ctx context.Context, coll *mongo.Collection, filter any, pagination pagination.Pagination, opts ...*options.FindOptions) ([]T, error) {
	skip := int64(pagination.GetPageNumber()-1) * int64(pagination.GetShowNumber())
	if skip < 0 || pagination.GetShowNumber() <= 0 {
		return nil, nil
	}
	opt := options.Find().SetSkip(skip).SetLimit(int64(pagination.GetShowNumber()))
	return Find[T](ctx, coll, filter, append(opts, opt)...)
}

func Count(ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.CountOptions) (int64, error) {
	count, err := coll.CountDocuments(ctx, filter, opts...)
	if err != nil {
		return 0, errs.WrapMsg(err, "mongo count")
	}
	return count, nil
}

func Exist(ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.CountOptions) (bool, error) {
	opts = append(opts, options.Count().SetLimit(1))
	count, err := Count(ctx, coll, filter, opts...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteOne(ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.DeleteOptions) error {
	if _, err := coll.DeleteOne(ctx, filter, opts...); err != nil {
		return errs.WrapMsg(err, "mongo delete one")
	}
	return nil
}

func DeleteOneResult(ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	res, err := coll.DeleteOne(ctx, filter, opts...)
	if err != nil {
		return nil, errs.WrapMsg(err, "mongo delete one")
	}
	return res, nil
}

func DeleteMany(ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.DeleteOptions) error {
	if _, err := coll.DeleteMany(ctx, filter, opts...); err != nil {
		return errs.WrapMsg(err, "mongo delete many")
	}
	return nil
}

func DeleteManyResult(ctx context.Context, coll *mongo.Collection, filter any, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	res, err := coll.DeleteMany(ctx, filter, opts...)
	if err != nil {
		return nil, errs.WrapMsg(err, "mongo delete many")
	}
	return res, nil
}

func Aggregate[T any](ctx context.Context, coll *mongo.Collection, pipeline any, opts ...*options.AggregateOptions) ([]T, error) {
	cur, err := coll.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		return nil, errs.WrapMsg(err, "mongo aggregate")
	}
	defer cur.Close(ctx)
	return Decodes[T](ctx, cur)
}

func Decodes[T any](ctx context.Context, cur *mongo.Cursor) ([]T, error) {
	var res []T
	if basic[T]() {
		var temp []map[string]T
		if err := cur.All(ctx, &temp); err != nil {
			return nil, errs.WrapMsg(err, "mongo decodes")
		}
		res = make([]T, 0, len(temp))
		for _, m := range temp {
			if len(m) != 1 {
				return nil, errs.ErrInternalServer.WrapMsg("mongo find result len(m) != 1")
			}
			for _, t := range m {
				res = append(res, t)
			}
		}
	} else {
		if err := cur.All(ctx, &res); err != nil {
			return nil, errs.WrapMsg(err, "mongo all")
		}
	}
	return res, nil
}

func DecodeOne[T any](decoder func(v any) error) (res T, err error) {
	if basic[T]() {
		var temp map[string]T
		if err = decoder(&temp); err != nil {
			err = errs.WrapMsg(err, "mongo decodes one")
			return
		}
		if len(temp) != 1 {
			err = errs.ErrInternalServer.WrapMsg("mongo find result len(m) != 1")
			return
		}
		for k := range temp {
			res = temp[k]
		}
	} else {
		if err = decoder(&res); err != nil {
			err = errs.WrapMsg(err, "mongo decoder")
			return
		}
	}
	return
}

func Ignore[T any](_ T, err error) error {
	return err
}

func IgnoreWarp[T any](_ T, err error) error {
	if err != nil {
		return errs.Wrap(err)
	}
	return err
}

func IncrVersion(dbs ...func() error) error {
	for _, fn := range dbs {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}
