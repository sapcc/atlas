package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name        string                 `yaml:"name"`
	Version     string                 `yaml:"version"`
	Discoveries map[string]interface{} `yaml:"discoveries"`
}

func GetConfig(opts Options) (cfg Config, err error) {
	if opts.ConfigFilePath == "" {
		return cfg, nil
	}
	yamlBytes, err := ioutil.ReadFile(opts.ConfigFilePath)
	if err != nil {
		return cfg, fmt.Errorf("read file file: %s", err.Error())
	}
	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parse config file: %s", err.Error())
	}

	return cfg, nil
}
