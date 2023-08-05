package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		Bot       `yaml:"bot"`
		Atlassian `yaml:"atlassian"`
	}
	Bot struct {
		Prefix string `yaml:"commandPrefix"`
		Token  string `yaml:"token"`
	}
	Atlassian struct {
		JiraUrl       string `yaml:"jiraUrl"`
		ConfluenceUrl string `yaml:"confluenceUrl"`
		Username      string `yaml:"username"`
		Password      string `yaml:"password"`
	}
)

func NewConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	err = yaml.Unmarshal(data, conf)
	return conf, err
}
