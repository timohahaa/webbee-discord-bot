package main

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Bot       `yaml:"bot"`
		Atlassian `yaml:"atlassian"`
	}
	Bot struct {
		Prefix string `yaml:"commandPrefix"`
		Token  string `yaml:"token" env:"DISCORD_TOKEN" env-required:"true"`
	}
	Atlassian struct {
		JiraUrl       string `yaml:"jiraUrl"`
		ConfluenceUrl string `yaml:"confluenceUrl"`
		Username      string `yaml:"username" env:"ATLASSIAN_USERNAME" env-required:"true"`
		Password      string `yaml:"password" env:"ATLASSIAN_PASSWORD" env-required:"true"`
	}
)

func NewConfig(path string) (*Config, error) {
	var conf = &Config{}
	err := cleanenv.ReadConfig("config.yaml", conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
