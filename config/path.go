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
