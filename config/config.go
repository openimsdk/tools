package config

import (
	"os"
	"path/filepath"

	"github.com/openimsdk/tools/errs"
	"gopkg.in/yaml.v2"
)

// ConfigLoader is responsible for loading configuration files.
type ConfigLoader struct {
	PathResolver PathResolver
}

func NewConfigLoader(pathResolver PathResolver) *ConfigLoader {
	return &ConfigLoader{PathResolver: pathResolver}
}

func (c *ConfigLoader) InitConfig(config any, configName, configFolderPath string) error {
	configFolderPath, err := c.resolveConfigPath(configName, configFolderPath)
	if err != nil {
		return errs.WrapMsg(err, "failed to resolve config path", "configName", configName)
	}

	data, err := os.ReadFile(configFolderPath)
	if err != nil {
		return errs.WrapMsg(err, "failed to read config file", "configName", configName)
	}

	if err = yaml.Unmarshal(data, config); err != nil {
		return errs.WrapMsg(err, "failed to unmarshal config data", "configName", configName)
	}

	return nil
}

func (c *ConfigLoader) resolveConfigPath(configName, configFolderPath string) (string, error) {
	if configFolderPath == "" {
		var err error
		configFolderPath, err = c.PathResolver.GetDefaultConfigPath()
		if err != nil {
			return "", errs.WrapMsg(err, "failed to get default config path", "configName", configName)
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
