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

//func Test_connectWithRetry(t *testing.T) {
//	type args struct {
//		ctx         context.Context
//		clientOpts  *options.ClientOptions
//		maxRetry    int
//		connTimeout time.Duration
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *mongo.Client
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := connectWithRetry(tt.args.ctx, tt.args.clientOpts, tt.args.maxRetry, tt.args.connTimeout)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("connectWithRetry() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("connectWithRetry() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestCheckMongo(t *testing.T) {
//	type args struct {
//		ctx    context.Context
//		config *Config
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := CheckMongo(tt.args.ctx, tt.args.config); (err != nil) != tt.wantErr {
//				t.Errorf("CheckMongo() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
