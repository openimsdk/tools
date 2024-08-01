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

import "gopkg.in/yaml.v2"

// Parser Configures the parser interface.
type Parser interface {
	Parse(data []byte, out any) error
}

// YAMLParser Configuration parser in YAML format.
type YAMLParser struct{}

func (y *YAMLParser) Parse(data []byte, out any) error {
	return yaml.Unmarshal(data, out)
}
