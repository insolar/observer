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
	"net/url"
	"strings"
	"time"

	insconf "github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/log"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/configuration/insconfig"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/dbconn"
	"github.com/insolar/observer/internal/models"
)

// CMCUrl holds cmc-api url
const CMCUrl = "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest"

var cmcAPIToken = flag.String("cmc-token", "", "api token for coin market cap")
var symbol = flag.String("symbol", "", "symbol for fetching from coin market cap stats")

func main() {
	params := insconfig.Params{
		ConfigStruct: configuration.Configuration{},
		EnvPrefix:    "observer",
	}
	insConfigurator := insconfig.NewInsConfigurator(params, insconfig.DefaultConfigPathGetter{
		GoFlags: flag.CommandLine,
	})
	parsedConf, err := insConfigurator.Load()
	if err != nil {
		panic(err)
	}
	cfg := parsedConf.(*configuration.Configuration)
	insConfigurator.PrintConfig(cfg)

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
		Price:                stats.Data.Info.Quote.USD.Price,
		PercentChange24Hours: stats.Data.Info.Quote.USD.PercentChange24Hours,
		Rank:                 stats.Data.Info.Rank,
		MarketCap:            stats.Data.Info.Quote.USD.MarketCap,
		Volume24Hours:        stats.Data.Info.Quote.USD.Volume24Hours,
		CirculatingSupply:    stats.Data.Info.CirculatingSupply,
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

func getStats(token string, symbol string, logger insolar.Logger) *CMCResponse {
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

	cmcResp := &CMCResponse{}
	err = json.Unmarshal(respBody, cmcResp)
	if err != nil {
		logger.Fatal(errors.Wrap(err, "failed to unmarshal body"))
	}

	logger.Debugf("response - %#v", cmcResp)

	if resp.StatusCode != http.StatusOK {
		logger.Fatalf("request failed with %v", cmcResp.Status)
	}

	return cmcResp
}

type CMCResponse struct {
	Data struct {
		Info *struct {
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
		} `json:"INS"`
	} `json:"data"`
	Status *struct {
		Timestamp    time.Time `json:"timestamp"`
		ErrorCode    int       `json:"error_code"`
		ErrorMessage string    `json:"error_message"`
	} `json:"status"`
}
