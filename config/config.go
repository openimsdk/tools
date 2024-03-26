package config

import (
	"os"
	"path/filepath"

	"github.com/openimsdk/tools/errs"
	"gopkg.in/yaml.v2"
)

// Loader is responsible for loading configuration files.
type Loader struct {
	PathResolver PathResolver
}

func NewLoader(pathResolver PathResolver) *Loader {
	return &Loader{PathResolver: pathResolver}
}

func (c *Loader) InitConfig(config any, configName, configFolderPath string) error {
	configFolderPath, err := c.resolveConfigPath(configName, configFolderPath)
	if err != nil {
		return errs.WrapMsg(err, "resolveConfigPath failed", "configName", configName, "configFolderPath", configFolderPath)
	}

	data, err := os.ReadFile(configFolderPath)
	if err != nil {
		return errs.WrapMsg(err, "ReadFile failed", "configFolderPath", configFolderPath)
	}

	if err = yaml.Unmarshal(data, config); err != nil {
		return errs.WrapMsg(err, "failed to unmarshal config data", "configName", configName)
	}

	return nil
}

func (c *Loader) resolveConfigPath(configName, configFolderPath string) (string, error) {
	if configFolderPath == "" {
		var err error
		configFolderPath, err = c.PathResolver.GetDefaultConfigPath()
		if err != nil {
			return "", errs.WrapMsg(err, "GetDefaultConfigPath failed", "configName", configName)
		}
	}

	configFilePath := filepath.Join(configFolderPath, configName)
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// Attempt to load from project root if not found in specified path
		projectRoot, err := c.PathResolver.GetProjectRoot()
		if err != nil {
			return "", err
		}
		configFilePath = filepath.Join(projectRoot, "config", configName)
	}
	return configFilePath, nil
}
