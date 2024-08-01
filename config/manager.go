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

type Manager struct {
	sources []ConfigSource
	parser  Parser
}

func NewManager(parser Parser) *Manager {
	return &Manager{
		parser: parser,
	}
}

func (cm *Manager) AddSource(source ConfigSource) {
	cm.sources = append(cm.sources, source)
}

func (cm *Manager) Load(config any) error {
	for _, source := range cm.sources {
		if data, err := source.Read(); err == nil {
			if err := cm.parser.Parse(data, config); err != nil {
				return err
			}
			break
		}
	}
	return nil
}
