package config

import (
	"os"

	"github.com/openimsdk/tools/errs"
)

// ConfigSource configuring source interfaces
type ConfigSource interface {
	Read() ([]byte, error)
}

// EnvVarSource read a configuration from an environment variable
type EnvVarSource struct {
	VarName string
}

func (e *EnvVarSource) Read() ([]byte, error) {
	value, exists := os.LookupEnv(e.VarName)
	if !exists {
		return nil, errs.New("environment variable not set")
	}
	return []byte(value), nil
}

// FileSystemSource read a configuration from a file
type FileSystemSource struct {
	FilePath string
}

func (f *FileSystemSource) Read() ([]byte, error) {
	return os.ReadFile(f.FilePath)
}
