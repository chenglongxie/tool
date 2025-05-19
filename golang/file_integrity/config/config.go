package config

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type Config struct {
	HostIP string `yaml:"host_ip"`
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
}

var cfg Config

func LoadConfig(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	return nil
}

func Get() *Config {
	return &cfg
}