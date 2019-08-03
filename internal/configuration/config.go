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
)

type Configurator interface {
	Default() *Configuration
}

func New() Configurator {
	return &Configuration{}
}

type Configuration struct {
	HTTPRouter HTTPRouter
	Replicator Replicator
}

func (c *Configuration) Default() *Configuration {
	return &Configuration{
		HTTPRouter: HTTPRouter{
			Addr: ":8080",
		},
		Replicator: Replicator{
			Addr:            "127.0.0.1:5678",
			MaxTransportMsg: 1073741824,
			RequestDelay:    10 * time.Second,
			BatchSize:       1000,
		},
	}
}

func Start() {

}
