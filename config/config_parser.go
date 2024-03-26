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
