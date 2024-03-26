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

package config

import (
	"os"
	"path/filepath"

	"github.com/openimsdk/tools/errs"
)

// PathResolver defines methods for resolving paths related to the application.
type PathResolver interface {
	GetDefaultConfigPath() (string, error)
	GetProjectRoot() (string, error)
}

type defaultPathResolver struct{}

// NewPathResolver creates a new instance of the default path resolver.
func NewPathResolver() *defaultPathResolver {
	return &defaultPathResolver{}
}

func (d *defaultPathResolver) GetDefaultConfigPath(relativePath string) (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", errs.WrapMsg(err, "Executable failed")
	}

	configPath := filepath.Join(filepath.Dir(executablePath), relativePath)
	return configPath, nil
}

// GetProjectRoot returns the project's root directory based on the relative depth specified.
// The depth parameter specifies how many levels up from the directory of the executable the project root is located.
func (d *defaultPathResolver) GetProjectRoot(depth int) (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", errs.WrapMsg(err, "Executable failed")
	}

	// Move up the specified number of directories to find the project root
	projectRoot := executablePath
	for i := 0; i < depth; i++ {
		projectRoot = filepath.Dir(projectRoot)
	}

	return projectRoot, nil
}
