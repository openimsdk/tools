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

func (cm *Manager) Load(config interface{}) error {
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
