// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package configuration

import (
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type APIConfig interface {
	GetListen() string
	GetDB() DB
	GetLog() Log
	GetFeeAmount() *big.Int
	GetPriceOrigin() string
	GetPrice() string
	GetCMCMarketStatsParams() CMCMarketStatsParamsEnabled
}

type CMCMarketStatsParamsEnabled struct {
	CirculatingSupply bool
	DailyChange       bool
	MarketCap         bool
	Rank              bool
	Volume            bool
}

type API struct {
	Listen string
	DB     DB
	Log    Log
}

func (API) Default() *API {
	return &API{
		Listen: ":0",
		DB: DB{
			URL:             "postgres://postgres@localhost/postgres?sslmode=disable",
			Attempts:        5,
			AttemptInterval: 3 * time.Second,
		},
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

func (a API) GetListen() string {
	return a.Listen
}

func (a API) GetDB() DB {
	return a.DB
}

func (a API) GetLog() Log {
	return a.Log
}

func (a API) GetFeeAmount() *big.Int {
	panic("shouldn't be implemented for the type API")
}

func (a API) GetPriceOrigin() string {
	panic("shouldn't be implemented for the type API")
}

func (a API) GetPrice() string {
	panic("shouldn't be implemented for the type API")
}

func (a API) GetCMCMarketStatsParams() CMCMarketStatsParamsEnabled {
	panic("shouldn't be implemented for the type API")
}
