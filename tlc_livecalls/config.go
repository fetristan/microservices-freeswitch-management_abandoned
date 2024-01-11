package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LiveCalls struct {
		Cycle          int    `yaml:"cycle"`
		Pole           string `yaml:"pole"`
		UrlApi         string `yaml:"url_api"`
		UrlApiUnmasked string `yaml:"url_api_unmasked"`
	} `yaml:"livecalls"`
	GrcpSessions struct {
		Host    string `yaml:"host"`
		Port    string `yaml:"port"`
		Timeout int    `yaml:"timeout"`
	} `yaml:"grcp_sessions"`
	GrcpDispatcher struct {
		Host    string `yaml:"host"`
		Port    string `yaml:"port"`
		Timeout int    `yaml:"timeout"`
	} `yaml:"grcp_dispatcher"`
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
