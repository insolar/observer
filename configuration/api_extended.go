// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build extended

package configuration

import (
	"math/big"
	"time"
)

type ApiExtended struct {
	Api `mapstructure:",squash"`

	FeeAmount            *big.Int
	PriceOrigin          string
	Price                string
	CMCMarketStatsParams CMCMarketStatsParamsEnabled
}

func (ApiExtended) Default() *ApiExtended {
	return &ApiExtended{
		Api: Api{
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
		},
		FeeAmount:   big.NewInt(1000000000),
		Price:       "0.05",
		PriceOrigin: "const", // const|binance|coin_market_cap
		CMCMarketStatsParams: CMCMarketStatsParamsEnabled{
			CirculatingSupply: true,
			DailyChange:       true,
			MarketCap:         true,
			Rank:              true,
			Volume:            true,
		},
	}
}

func (a ApiExtended) GetListen() string {
	return a.Listen
}

func (a ApiExtended) GetDB() DB {
	return a.DB
}

func (a ApiExtended) GetLog() Log {
	return a.Log
}

func (a ApiExtended) GetFeeAmount() *big.Int {
	return a.FeeAmount
}

func (a ApiExtended) GetPriceOrigin() string {
	return a.PriceOrigin
}

func (a ApiExtended) GetPrice() string {
	return a.Price
}

func (a ApiExtended) GetCMCMarketStatsParams() CMCMarketStatsParamsEnabled {
	return a.CMCMarketStatsParams
}

func GetApiConfig() ApiConfig {
	return &ApiExtended{}
}
