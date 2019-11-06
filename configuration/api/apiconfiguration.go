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

package api

import (
	"math/big"

	"github.com/sirupsen/logrus"

	"time"

	"github.com/insolar/observer/configuration"
)

type Configuration struct {
	API       configuration.API
	DB        configuration.DB
	FeeAmount *big.Int
	LogLevel  string
}

func Default() *Configuration {
	return &Configuration{
		API: configuration.API{
			Addr: ":0",
		},
		DB: configuration.DB{
			URL:             "postgres://postgres@localhost/postgres?sslmode=disable",
			Attempts:        5,
			AttemptInterval: 3 * time.Second,
			CreateTables:    false,
		},
		LogLevel:  logrus.DebugLevel.String(),
		FeeAmount: big.NewInt(1000000000),
	}
}
