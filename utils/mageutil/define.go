package mageutil

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"runtime"
)

var (
	binaries           map[string]int
	toolBinaries       []string
	MaxFileDescriptors int
)

type Config struct {
	Binaries           map[string]int `yaml:"binaries"`
	ToolBinaries       []string       `yaml:"toolBinaries"`
	MaxFileDescriptors int            `yaml:"maxFileDescriptors"`
}

func init() {
	yamlFile, err := ioutil.ReadFile("start-config.yml")
	if err != nil {
		log.Fatalf("error reading YAML file: %v", err)
	}

	// 解析YAML
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("error unmarshalling YAML: %v", err)
	}

	adjustedBinaries := make(map[string]int)
	for binary, count := range config.Binaries {
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		adjustedBinaries[binary] = count
	}

	binaries = adjustedBinaries
	toolBinaries = config.ToolBinaries
	MaxFileDescriptors = config.MaxFileDescriptors
}
