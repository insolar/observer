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
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
)

type Configuration struct {
	Listen      string
	DB          configuration.DB
	FeeAmount   *big.Int
	PriceOrigin string
	Price       string
	Log         Log
}

func (c Configuration) GetConfig() interface{} {
	return &c
}

type Log struct {
	Level        string
	Format       string
	OutputType   string
	OutputParams string
	Buffer       int
}

func Default() *Configuration {
	return &Configuration{
		Listen: ":0",
		DB: configuration.DB{
			URL:             "postgres://postgres@localhost/postgres?sslmode=disable",
			Attempts:        5,
			AttemptInterval: 3 * time.Second,
		},
		FeeAmount:   big.NewInt(1000000000),
		Price:       "0.05",
		PriceOrigin: "const", //const|binance|coin_market_cap
		Log: Log{
			Level:        "debug",
			Format:       "text",
			OutputType:   "stderr",
			OutputParams: "",
			Buffer:       0,
		},
	}
}

func ToBigIntHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {

		if t != reflect.TypeOf(big.NewInt(0)) {
			return data, nil
		}

		switch f {
		case reflect.TypeOf(""):
			res := new(big.Int)
			if _, err := fmt.Sscan(data.(string), res); err != nil {
				return data, errors.Wrapf(err, "failed to parse big.Int, input %v", data)
			}
			return res, nil
		case reflect.TypeOf(0):
			return big.NewInt(int64(data.(int))), nil
		}
		return data, nil
	}
}
