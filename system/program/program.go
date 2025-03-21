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

package program

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ExitWithError(err error) {
	progName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "%s exit -1: %+v\n", progName, err)
	os.Exit(-1)
}

func SIGTERMExit() {
	progName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Warning %s receive process terminal SIGTERM exit 0\n", progName)
}

// GetProcessName retrieves the name of the currently running process.
// It achieves this by parsing os.Args[0], which typically contains the full path to the program.
// If os.Args[0] is empty or unset for some reason, the function returns an empty string.
// Note: This function assumes that os.Args contains at least the program name. This is a safe assumption under normal circumstances.
func GetProcessName() string {
	args := os.Args
	if len(args) > 0 {
		segments := strings.Split(args[0], "/")
		if len(segments) > 0 {
			return segments[len(segments)-1]
		}
	}
	return ""
}
