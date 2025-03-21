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

package splitter

import (
	"reflect"
	"testing"
)

func TestNewSplitter(t *testing.T) {
	data := []string{"a", "b", "c", "d", "e", "f"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	if splitter.splitCount != splitCount {
		t.Errorf("Expected splitCount %d, got %d", splitCount, splitter.splitCount)
	}
	if !reflect.DeepEqual(splitter.data, data) {
		t.Errorf("Expected data %v, got %v", data, splitter.data)
	}
}

func TestGetSplitResult_EvenSplit(t *testing.T) {
	data := []string{"a", "b", "c", "d"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	expected := []*SplitResult{
		{Item: []string{"a", "b"}},
		{Item: []string{"c", "d"}},
	}
	result := splitter.GetSplitResult()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %v, got %v", expected, result)
	}
}

func TestGetSplitResult_OddSplit(t *testing.T) {
	data := []string{"a", "b", "c", "d", "e"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	expected := []*SplitResult{
		{Item: []string{"a", "b"}},
		{Item: []string{"c", "d"}},
		{Item: []string{"e"}},
	}
	result := splitter.GetSplitResult()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %v, got %v", expected, result)
	}
}

func TestGetSplitResult_SingleElement(t *testing.T) {
	data := []string{"a"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	expected := []*SplitResult{{Item: []string{"a"}}}
	result := splitter.GetSplitResult()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %v, got %v", expected, result)
	}
}
