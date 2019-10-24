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
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type API struct {
	Addr string
}

type APIConfiguration struct {
	API API
	DB  DB
}

func APIDefault() *APIConfiguration {
	return &APIConfiguration{
		API: API{
			Addr: ":0",
		},
		DB: DB{
			URL:             "postgres://postgres@localhost/postgres?sslmode=disable",
			Attempts:        5,
			AttemptInterval: 3 * time.Second,
			CreateTables:    false,
		},
	}
}

func APILoad() *APIConfiguration {
	printWorkingDir()
	actual := apiload()
	printConfig(actual)
	return actual
}

func apiload() *APIConfiguration {
	v := viper.New()
	v.SetConfigName(APIConfigName)
	v.SetConfigType(ConfigType)
	v.AddConfigPath(".")
	v.AddConfigPath(".artifacts")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Warnf("config file not found (file=%v). Default configuration is used", APIConfigFilePath)
		} else {
			log.Error(errors.Wrapf(err, "failed to load config. Default configuration is used"))
		}
		return APIDefault()
	}
	actual := &APIConfiguration{}
	err := v.Unmarshal(actual)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to unmarshal readed from file config into configuration structure. Default configuration is used"))
		return APIDefault()
	}
	return actual
}

func apicleanSecrects(c *APIConfiguration) (*APIConfiguration, error) {
	buf, err := insolar.Serialize(c)
	if err != nil {
		return nil, errors.New("failed to serialize config")
	}
	cc := &APIConfiguration{}
	if err := insolar.Deserialize(buf, cc); err != nil {
		return nil, errors.New("failed to deserialize config")
	}
	cc.DB.URL = replacePassword(cc.DB.URL)
	return cc, nil
}
