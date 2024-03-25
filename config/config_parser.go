package config

import "gopkg.in/yaml.v2"

// ConfigParser Configures the parser interface
type ConfigParser interface {
	Parse(data []byte, out interface{}) error
}

// YAMLParser Configuration parser in YAML format
type YAMLParser struct{}

func (y *YAMLParser) Parse(data []byte, out interface{}) error {
	return yaml.Unmarshal(data, out)
}
