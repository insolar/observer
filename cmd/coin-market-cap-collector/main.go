// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/insolar/insconfig"
	insconf "github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/log"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/dbconn"
	"github.com/insolar/observer/internal/models"
)

// CMCUrl holds cmc-api url
const CMCUrl = "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest"

var cmcAPIToken = flag.String("cmc-token", "", "api token for coin market cap")
var symbol = flag.String("symbol", "", "symbol for fetching from coin market cap stats")

func main() {
	cfg := &configuration.CollectorCoinMarketCap{}
	params := insconfig.Params{
		EnvPrefix: "coin-market-cap-collector",
		ConfigPathGetter: &insconfig.FlagPathGetter{
			GoFlags: flag.CommandLine,
		},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(cfg); err != nil {
		panic(err)
	}
	insConfigurator.ToYaml(cfg)

	if cmcAPIToken == nil || len(*cmcAPIToken) == 0 || len(strings.TrimSpace(*cmcAPIToken)) == 0 {
		panic("cmc-token should be provided")
	}
	if symbol == nil || len(*symbol) == 0 || len(strings.TrimSpace(*symbol)) == 0 {
		panic("symbol should be provided")
	}

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
	repo := postgres.NewCoinMarketCapStatsRepository(db)

	logger.Info("start fetching from cmc-api")
	stats := getStats(*cmcAPIToken, *symbol, logger)
	err = repo.InsertStats(&models.CoinMarketCapStats{
		Price:                stats.Quote.USD.Price,
		PercentChange24Hours: stats.Quote.USD.PercentChange24Hours,
		Rank:                 stats.Rank,
		MarketCap:            stats.Quote.USD.MarketCap,
		Volume24Hours:        stats.Quote.USD.Volume24Hours,
		CirculatingSupply:    stats.CirculatingSupply,
		Created:              time.Now().UTC(),
	})
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("finish fetching from cmc-api successfully")
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

func getStats(token string, symbol string, logger insolar.Logger) *CmcInfo {
	logger.Info("getStats for symbol %v", symbol)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", CMCUrl, nil)
	if err != nil {
		logger.Fatal(err)
	}

	q := url.Values{}
	q.Add("symbol", symbol)
	q.Add("convert", "USD")

	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", token)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		logger.Fatal(errors.Wrap(err, "failed to send request"))
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Fatal(errors.Wrap(err, "can't read the response body"))
	}
	logger.Debugf("response body - %v", string(respBody))

	if resp.StatusCode != http.StatusOK {
		logger.Fatalf("request failed. %v", string(respBody))
	}

	var respMap map[string]*json.RawMessage
	err = json.Unmarshal(respBody, &respMap)
	if err != nil {
		logger.Fatal(errors.Wrap(err, "can't parse the response body"))
	}
	err = json.Unmarshal(*respMap["data"], &respMap)
	if err != nil {
		logger.Fatal(errors.Wrap(err, "can't parse the data field"))
	}

	info := &CmcInfo{}
	err = json.Unmarshal(*respMap[symbol], &info)
	if err != nil {
		logger.Fatal(errors.Wrap(err, "can't parse symbol field into struct"))
	}

	logger.Debugf("response - %#v", info)

	return info
}

type CmcInfo struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	Slug              string    `json:"slug"`
	Rank              int       `json:"cmc_rank"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TotalSupply       float64   `json:"total_supply"`
	MaxSupply         float64   `json:"max_supply"`
	LastUpdated       time.Time `json:"last_updated"`
	DateAdded         time.Time `json:"date_added"`
	Quote             struct {
		USD struct {
			Price                float64   `json:"price"`
			Volume24Hours        float64   `json:"volume_24h"`
			PercentChange1Hour   float64   `json:"percent_change_1h"`
			PercentChange24Hours float64   `json:"percent_change_24h"`
			PercentChange7Days   float64   `json:"percent_change_7d"`
			MarketCap            float64   `json:"market_cap"`
			LastUpdated          time.Time `json:"last_updated"`
		} `json:"usd"`
	} `json:"quote"`
}
