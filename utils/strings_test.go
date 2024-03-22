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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntToString(t *testing.T) {
	assert.Equal(t, "123", IntToString(123))
}

func TestStringToInt(t *testing.T) {
	assert.Equal(t, 123, StringToInt("123"))
}

func TestStringToInt64(t *testing.T) {
	assert.Equal(t, int64(123), StringToInt64("123"))
}

func TestStringToInt32(t *testing.T) {
	assert.Equal(t, int32(123), StringToInt32("123"))
}

func TestInt32ToString(t *testing.T) {
	assert.Equal(t, "123", Int32ToString(123))
}

func TestUint32ToString(t *testing.T) {
	assert.Equal(t, "123", Uint32ToString(123))
}

func TestIsContain(t *testing.T) {
	list := []string{"apple", "banana", "cherry"}
	assert.True(t, IsContain("banana", list))
	assert.False(t, IsContain("date", list))
}

func TestIsContainInt32(t *testing.T) {
	list := []int32{1, 2, 3}
	assert.True(t, IsContainInt32(2, list))
	assert.False(t, IsContainInt32(4, list))
}

func TestIsContainInt(t *testing.T) {
	list := []int{1, 2, 3}
	assert.True(t, IsContainInt(2, list))
	assert.False(t, IsContainInt(4, list))
}

func TestRemoveDuplicateElement(t *testing.T) {
	idList := []string{"a", "b", "a", "c", "b"}
	expected := []string{"a", "b", "c"}
	result := RemoveDuplicateElement(idList)
	assert.ElementsMatch(t, expected, result)
}

func TestIsDuplicateStringSlice(t *testing.T) {
	assert.False(t, IsDuplicateStringSlice([]string{"a", "b", "c"}))
	assert.True(t, IsDuplicateStringSlice([]string{"a", "b", "a"}))
}
