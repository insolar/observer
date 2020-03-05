// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package api

import (
	"math/big"
	"time"

	"github.com/insolar/observer/configuration"
)

type Configuration struct {
	Listen               string
	DB                   configuration.DB
	FeeAmount            *big.Int
	PriceOrigin          string
	Price                string
	CMCMarketStatsParams CMCMarketStatsParamsEnabled
	Log                  Log
}

type Log struct {
	Level        string
	Format       string
	OutputType   string
	OutputParams string
	Buffer       int
}

type CMCMarketStatsParamsEnabled struct {
	CirculatingSupply bool
	DailyChange       bool
	MarketCap         bool
	Rank              bool
	Volume            bool
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
		CMCMarketStatsParams: CMCMarketStatsParamsEnabled{
			CirculatingSupply: true,
			DailyChange:       true,
			MarketCap:         true,
			Rank:              true,
			Volume:            true,
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
