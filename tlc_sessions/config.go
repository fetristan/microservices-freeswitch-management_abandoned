package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sessions struct {
		Cycle int `yaml:"cycle"`
	} `yaml:"sessions"`
	Freeswitch []struct {
		Host        string `yaml:"host"`
		Port        string `yaml:"port"`
		Pass        string `yaml:"pass"`
		Pole        string `yaml:"pole"`
		RetryNumber int    `yaml:"retry_number"`
	} `yaml:"freeswitch"`
	Database struct {
		Host   string `yaml:"host"`
		Port   string `yaml:"port"`
		User   string `yaml:"user"`
		Pass   string `yaml:"pass"`
		Dbname string `yaml:"dbname"`
	} `yaml:"database"`
	Redis struct {
		Host   string `yaml:"host"`
		Port   string `yaml:"port"`
		User   string `yaml:"user"`
		Pass   string `yaml:"pass"`
		Dbname int    `yaml:"db"`
	} `yaml:"redis"`
	GrcpListener struct {
		Port int `yaml:"port"`
	} `yaml:"grcp_listener"`
}

func readConf(filename string) (*Config, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
