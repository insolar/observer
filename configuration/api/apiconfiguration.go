// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

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
	Listen    string
	DB        configuration.DB
	FeeAmount *big.Int
	Log       Log
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
		FeeAmount: big.NewInt(1000000000),
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
