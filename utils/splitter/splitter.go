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

// SplitResult holds a slice of strings as a result of the splitting operation.
type SplitResult struct {
	Item []string
}

// Splitter is responsible for splitting a slice of strings into multiple parts.
type Splitter struct {
	splitCount int      // The number of parts to split the data into.
	data       []string // The original data to be split.
}

// NewSplitter creates a new Splitter instance with the specified split count and data.
func NewSplitter(splitCount int, data []string) *Splitter {
	return &Splitter{splitCount: splitCount, data: data}
}

// GetSplitResult performs the splitting operation and returns the results as a slice of SplitResult.
// Each SplitResult contains a slice of strings, representing a part of the original data.
func (s *Splitter) GetSplitResult() (result []*SplitResult) {
	remain := len(s.data) % s.splitCount
	integer := len(s.data) / s.splitCount

	for i := 0; i < integer; i++ {
		r := new(SplitResult)
		r.Item = s.data[i*s.splitCount : (i+1)*s.splitCount]
		result = append(result, r)
	}
	if remain > 0 {
		r := new(SplitResult)
		r.Item = s.data[integer*s.splitCount:]
		result = append(result, r)
	}
	return result
}
