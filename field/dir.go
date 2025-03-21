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

package field

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openimsdk/tools/errs"
)

// OutDir creates the absolute path name from path and checks if the path exists and is a directory.
// Returns absolute path including trailing '/' or error if the path does not exist or is not a directory.
func OutDir(path string) (string, error) {
	outDir, err := filepath.Abs(path)
	if err != nil {
		return "", errs.WrapMsg(err, "failed to resolve absolute path", "path", path)
	}

	stat, err := os.Stat(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errs.WrapMsg(err, "output directory does not exist", "path", outDir)
		}
		return "", errs.WrapMsg(err, "failed to stat output directory", "path", outDir)
	}

	if !stat.IsDir() {
		return "", errs.WrapMsg(fmt.Errorf("specified path %s is not a directory", outDir), "specified path is not a directory", "path", outDir)
	}
	outDir += "/"
	return outDir, nil
}
