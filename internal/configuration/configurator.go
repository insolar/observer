//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package configuration

import (
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	ConfigName     = "observer"
	ConfigType     = "yaml"
	ConfigFilePath = ConfigName + "." + ConfigType
)

type Configurator interface {
	Actual() *Configuration
}

func Load() Configurator {
	printWorkingDir()
	c := &configurator{actual: load()}
	printConfig(c.actual)
	return c
}

func load() *Configuration {
	v := viper.New()
	v.SetConfigName(ConfigName)
	v.SetConfigType(ConfigType)
	v.AddConfigPath(".")
	v.AddConfigPath(".artifacts")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Warnf("config file not found (file=%v). Default configuration is used", ConfigFilePath)
		} else {
			log.Error(errors.Wrapf(err, "failed to load config. Default configuration is used"))
		}
		return Default()
	}
	actual := &Configuration{}
	err := v.Unmarshal(actual)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to unmarshal readed from file config into configuration structure. Default configuration is used"))
		return Default()
	}
	return actual
}

type configurator struct {
	actual *Configuration
}

func (c *configurator) Actual() *Configuration {
	return c.actual
}

func printWorkingDir() {
	wd, _ := os.Getwd()
	log.Infof("Working dir: %s", wd)
}

func printConfig(c *Configuration) {
	out, err := yaml.Marshal(c)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to marshal default config structure"))
	}
	log.Infof("Loaded configuration: \n %s \n", string(out))
}
