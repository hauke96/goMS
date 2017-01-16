package services

import (
	"encoding/json"
	"goMS/src/technical/material"
	"goMS/src/technical/services/logger"
	"io/ioutil"
)

// simplify the way of access to structs
type Config material.Config
type TopicConfig material.TopicConfig

type ConfigLoader struct {
	TopicConfig *TopicConfig
}

// LoadTopics reads the config file for the topics and fills the TopicConfig field
// The default location is /config/topics.json
func (cl *ConfigLoader) LoadTopics(filename string) {
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		logger.Fatal(err.Error())
	}

	cl.TopicConfig = &TopicConfig{}
	json.Unmarshal(data, cl.TopicConfig)
}
