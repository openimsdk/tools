// Copyright © 2025 OpenIM open source community. All rights reserved.
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

package redistask

type Option[T any, K comparable] func(*QueueManager[T, K])

func WithNamespace[T any, K comparable](namespace string) Option[T, K] {
	return func(m *QueueManager[T, K]) {
		m.namespace = namespace
	}
}

func WithEqualDataFunc[T any, K comparable](fn func(a, b T) bool) Option[T, K] {
	return func(m *QueueManager[T, K]) {
		m.equalDataFunc = fn
	}
}

func WithAfterProcessPushFunc[T any, K comparable](fn func(key K, data T)) Option[T, K] {
	return func(m *QueueManager[T, K]) {
		m.afterProcessPushFunc = append(m.afterProcessPushFunc, fn)
	}
}

func WithStrategy[T any, K comparable](s strategy) Option[T, K] {
	return func(m *QueueManager[T, K]) {
		m.assignStrategy = getStrategy[T, K](s)
	}
}

func WithMarshalFunc[T any, K comparable](fn func(T) ([]byte, error)) Option[T, K] {
	return func(m *QueueManager[T, K]) {
		m.marshalFunc = fn
	}
}

func WithUnmarshalFunc[T any, K comparable](fn func([]byte, *T) error) Option[T, K] {
	return func(m *QueueManager[T, K]) {
		m.unmarshalFunc = fn
	}
}
