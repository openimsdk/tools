package config

type ConfigManager struct {
	sources []ConfigSource
	parser  ConfigParser
}

func NewConfigManager(parser ConfigParser) *ConfigManager {
	return &ConfigManager{
		parser: parser,
	}
}

func (cm *ConfigManager) AddSource(source ConfigSource) {
	cm.sources = append(cm.sources, source)
}

func (cm *ConfigManager) Load(config interface{}) error {
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
