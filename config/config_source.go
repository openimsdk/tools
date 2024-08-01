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

	"github.com/openimsdk/tools/errs"
)

// ConfigSource configuring source interfaces.
type ConfigSource interface {
	Read() ([]byte, error)
}

// EnvVarSource read a configuration from an environment variable.
type EnvVarSource struct {
	VarName string
}

func (e *EnvVarSource) Read() ([]byte, error) {
	value, exists := os.LookupEnv(e.VarName)
	if !exists {
		return nil, errs.New("environment variable not set").Wrap()
	}
	return []byte(value), nil
}

// FileSystemSource read a configuration from a file.
type FileSystemSource struct {
	FilePath string
}

func (f *FileSystemSource) Read() ([]byte, error) {
	r, err := os.ReadFile(f.FilePath)
	return r, errs.WrapMsg(err, "ReadFile failed ", "FilePath", f.FilePath)
}
