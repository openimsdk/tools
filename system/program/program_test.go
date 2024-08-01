// Copyright © 2024 OpenIM. All rights reserved.
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

package program

import (
	"os"
	"testing"
)

// TestGetProcessName tests the GetProcessName function to ensure it returns the expected process name.
func TestGetProcessName(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	expected := "testprog"
	os.Args = []string{"/path/to/" + expected}

	got := GetProcessName()
	if got != expected {
		t.Errorf("GetProcessName() = %q, want %q", got, expected)
	}
}
