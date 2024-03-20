package config

type Redis struct {
	ClusterMode    bool     `yaml:"clusterMode"`
	Address        []string `yaml:"address"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	EnablePipeline bool     `yaml:"enablePipeline"`
}
