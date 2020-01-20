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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	insconf "github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/log"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/dbconn"
	"github.com/insolar/observer/internal/models"
	"github.com/pkg/errors"
)

// BinanceAPIUrl stores Binance url
const BinanceAPIUrl = "https://api.binance.com/api/v3/"

func main() {
	symbol := flag.String("symbol", "", "token symbol")
	flag.Parse()

	if symbol == nil || len(*symbol) == 0 || len(strings.TrimSpace(*symbol)) == 0 {
		panic("symbol should be provided")
	}

	cfg := configuration.Load()
	loggerConfig := insconf.Log{
		Level:      cfg.Log.Level,
		Formatter:  cfg.Log.Format,
		Adapter:    "zerolog",
		OutputType: "stderr",
		BufferSize: 0,
	}
	_, logger := initGlobalLogger(context.Background(), loggerConfig)
	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatal(err.Error())
	}

	btcPrice, err := getPrice(logger, "BTCUSDT")
	if err != nil {
		logger.Fatal(err)
	}

	symbolPrice, err := getPrice(logger, *symbol+"BTC")
	if err != nil {
		logger.Fatal(err)
	}

	symbolStats, err := getStats(logger, *symbol+"BTC")
	if err != nil {
		logger.Fatal(err)
	}

	err = insertStats(
		logger,
		postgres.NewBinanceStatsRepository(db),
		btcPrice,
		symbolPrice,
		symbolStats)
	if err != nil {
		logger.Fatal(err)
	}
}

func initGlobalLogger(ctx context.Context, cfg insconf.Log) (context.Context, insolar.Logger) {
	inslog, err := log.NewGlobalLogger(cfg)
	if err != nil {
		panic(err)
	}

	ctx = inslogger.SetLogger(ctx, inslog)
	log.SetGlobalLogger(inslog)

	return ctx, inslog
}

type price struct {
	Price string `json:"price"`
	Mins  int    `json:"mins"`
}

func getPrice(log insolar.Logger, symbol string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(BinanceAPIUrl + "avgPrice?symbol=" + symbol)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", errors.New("binance api request limits are exceeded")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	price := price{}
	err = json.Unmarshal(body, &price)
	if err != nil {
		return "", err
	}

	log.Debugf("get price for symbol:" + symbol + " result:" + string(body))

	return price.Price, nil
}

type symbolDayStat struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
}

func getStats(log insolar.Logger, symbol string) (symbolDayStat, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(BinanceAPIUrl + "ticker/24hr?symbol=" + symbol)
	if err != nil {
		return symbolDayStat{}, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return symbolDayStat{}, errors.New("binance api request limits are exceeded")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return symbolDayStat{}, err
	}
	stats := symbolDayStat{}
	err = json.Unmarshal(body, &stats)
	if err != nil {
		return symbolDayStat{}, err
	}

	log.Debugf("get 24hrs stats for symbol:" + symbol + " result:" + string(body))

	return stats, nil
}

func insertStats(
	log insolar.Logger,
	repo *postgres.BinanceStatsRepository,
	btcUsdtPrice string,
	symbolBtcPrice string,
	stats symbolDayStat,
) error {
	btcUsdtConverted, err := strconv.ParseFloat(btcUsdtPrice, 64)
	if err != nil {
		return errors.Wrap(err, "btc price can't be casted")
	}

	symbolBtcConverted, err := strconv.ParseFloat(symbolBtcPrice, 64)
	if err != nil {
		return errors.Wrap(err, "symbol price can't be casted")
	}

	symbolPriceUsd := btcUsdtConverted * symbolBtcConverted
	newStats := &models.BinanceStats{
		Symbol:             stats.Symbol,
		SymbolPriceBTC:     symbolBtcPrice,
		SymbolPriceUSD:     symbolPriceUsd,
		BTCPriceUSD:        btcUsdtPrice,
		PriceChangePercent: stats.PriceChangePercent,
	}

	log.Debugf("collected binance stats: %+v", newStats)
	return repo.InsertStats(newStats)
}
