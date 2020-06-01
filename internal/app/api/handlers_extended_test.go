// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build extended

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

func getApiConfig() configuration.ApiConfig {
	return configuration.ApiExtended{
		FeeAmount: testFee,
		Price:     testPrice,
	}
}

func TestMigrationAddresses_WrongArguments(t *testing.T) {
	// if `limit` is not a number, API returns `bad request`
	resp, err := http.Get("http://" + apihost + "/admin/migration/addresses?limit=LOL")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is zero, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=0")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is negative, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=-10")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `limit` is > 1000, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=1001")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// if `index` is not a number, API returns `bad request`
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=100&index=LOL")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestMigrationAddresses_HappyPath(t *testing.T) {
	defer truncateDB(t)

	// Make sure /admin/migration/addresses returns non-assigned migration addresses
	// sorted by ID with provided `limit` and `index` arguments.

	// insert migration addresses
	var err error
	wasted := []bool{false, false, true, false, true}
	for i := 0; i < len(wasted); i++ {
		migrationAddress := models.MigrationAddress{
			ID:        32000 + int64(i),
			Addr:      fmt.Sprintf("migration_addr_%v", i),
			Timestamp: time.Now().Unix(),
			Wasted:    wasted[i],
		}

		err = db.Insert(&migrationAddress)
		require.NoError(t, err)
	}

	// request two oldest non-assigned migration addresses
	resp, err := http.Get("http://" + apihost + "/admin/migration/addresses?limit=2")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received []map[string]string
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, 2, len(received))
	require.Equal(t, "32000", received[0]["index"])
	require.Equal(t, "migration_addr_0", received[0]["address"])
	require.Equal(t, "32001", received[1]["index"])
	require.Equal(t, "migration_addr_1", received[1]["address"])

	// request the rest of non-assigned migration addresses
	resp, err = http.Get("http://" + apihost + "/admin/migration/addresses?limit=100&index=" + received[1]["index"])
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, 1, len(received))
	require.Equal(t, "32003", received[0]["index"])
	require.Equal(t, "migration_addr_3", received[0]["address"])
}

func TestMigrationAddressesCount(t *testing.T) {
	defer truncateDB(t)

	// Make sure /admin/migration/addresses/count returns the total number
	// of non-assigned migration addresses.

	// insert migration addresses
	var err error
	wasted := []bool{true, false, true, false, true}
	expectedCount := 0
	for i := 0; i < len(wasted); i++ {
		migrationAddress := models.MigrationAddress{
			ID:        31000 + int64(i),
			Addr:      fmt.Sprintf("migration_addr_%v", i),
			Timestamp: time.Now().Unix(),
			Wasted:    wasted[i],
		}

		if !wasted[i] {
			expectedCount++
		}

		err = db.Insert(&migrationAddress)
		require.NoError(t, err)
	}

	resp, err := http.Get("http://" + apihost + "/admin/migration/addresses/count")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var received map[string]int
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)
	require.Equal(t, expectedCount, received["count"])
}

func TestObserverServer_SupplyStatsEmpty(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/stats/supply/total")
	require.NoError(t, err)
	require.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK)
}

func TestObserverServer_SupplyStats(t *testing.T) {
	total := "1111111111111"
	totalr := "111.1111111111"

	coins := models.SupplyStats{
		Created: time.Now(),
		Total:   total,
	}

	err := db.Insert(&coins)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/stats/supply/total")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, totalr, string(bodyBytes))
}

func TestFee(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/123")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesFeeYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, testFee.String(), received.Fee)
	})

	t.Run("uuid", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/31f277c7-67f8-45b5-ae26-ff127d62a9ba")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesInvalidAmountYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, []string{"invalid amount"}, received.Error)
	})

	t.Run("negative", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/-1")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesInvalidAmountYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, []string{"negative amount"}, received.Error)
	})
}

func TestObserverServer_NetworkStats(t *testing.T) {
	stats := models.NetworkStats{
		Created:           time.Now(),
		PulseNumber:       123,
		TotalTransactions: 23,
		MonthTransactions: 10,
		TotalAccounts:     3,
		Nodes:             11,
		CurrentTPS:        45,
		MaxTPS:            1498,
	}

	repo := postgres.NewNetworkStatsRepository(db)
	err := repo.InsertStats(stats)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/stats/network")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonResp := ResponsesNetworkStatsYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	expected := ResponsesNetworkStatsYaml{
		Accounts:              3,
		CurrentTPS:            45,
		LastMonthTransactions: 10,
		MaxTPS:                1498,
		Nodes:                 11,
		TotalTransactions:     23,
	}
	require.Equal(t, expected, jsonResp)
}

func TestObserverServer_MarketStats(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/stats/market")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	jsonResp := ResponsesMarketStatsYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	expected := ResponsesMarketStatsYaml{
		Price: "0.05",
	}
	require.Equal(t, expected, jsonResp)
}

func TestObserverServer_Notifications(t *testing.T) {
	apiUrl := "http://" + apihost + "/api/notification"

	// No content status is displayed
	if _, err := db.Exec("DELETE FROM notifications"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Present notification is published
	presentNotificationName := uuid.New().String()
	presentNotification := &models.Notification{
		Message: presentNotificationName,
		Start:   time.Now().Add(-3 * time.Hour),
		Stop:    time.Now().Add(3 * time.Hour),
	}
	err = db.Insert(presentNotification)
	require.NoError(t, err)

	resp, err = http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	jsonResp := ResponsesNotificationInfoYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	require.Equal(t, presentNotificationName, jsonResp.Notification)

	// Past notification was not published
	resp, err = http.Get(apiUrl)
	require.NoError(t, err)

	oldNotificationName := uuid.New().String()
	err = db.Insert(&models.Notification{
		Message: oldNotificationName,
		Start:   time.Now().Add(-10 * time.Hour),
		Stop:    time.Now().Add(-9 * time.Hour),
	})
	require.NoError(t, err)

	resp, err = http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	jsonResp = ResponsesNotificationInfoYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	require.NotEqual(t, oldNotificationName, jsonResp.Notification)

	// Future notification is not published yet
	futureNotificationName := uuid.New().String()
	err = db.Insert(&models.Notification{
		Message: futureNotificationName,
		Start:   time.Now().Add(20 * time.Hour),
		Stop:    time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	resp, err = http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	jsonResp = ResponsesNotificationInfoYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	require.NotEqual(t, futureNotificationName, jsonResp.Notification)
}

func TestIsMigrationAddressFailed(t *testing.T) {
	address := "0x012345678901234567890123456789qwertyuiop"
	migrationAddress := models.MigrationAddress{
		ID:        32000 + int64(0),
		Addr:      address,
		Timestamp: time.Now().Unix(),
		Wasted:    true,
	}

	err := db.Insert(&migrationAddress)
	require.NoError(t, err)

	type testCase struct {
		address string
		result  bool
	}

	var testCases []testCase
	testCases = append(
		testCases,
		testCase{address: notExistedMigrationAddress, result: false},
		testCase{address: address, result: true},
	)

	for _, test := range testCases {
		resp, err := http.Get("http://" + apihost + "/admin/isMigrationAddress/" + test.address)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		jsonResp := ResponsesIsMigrationAddressYaml{}
		err = json.Unmarshal(bodyBytes, &jsonResp)
		require.NoError(t, err)
		expected := ResponsesIsMigrationAddressYaml{
			IsMigrationAddress: test.result,
		}
		require.Equal(t, expected, jsonResp)
	}

}

func TestObserverServer_CMC_Price(t *testing.T) {
	// first interval
	statsTime := time.Date(2020, 1, 3, 6, 0, 0, 0, time.UTC)
	err := db.Insert(&models.CoinMarketCapStats{
		Price:                100,
		PercentChange24Hours: 1,
		Rank:                 2,
		MarketCap:            3,
		Volume24Hours:        4,
		CirculatingSupply:    5,
		Created:              statsTime,
	})
	require.NoError(t, err)

	statsTime = time.Date(2020, 1, 3, 7, 0, 0, 0, time.UTC)
	err = db.Insert(&models.CoinMarketCapStats{
		Price:                200,
		PercentChange24Hours: 11,
		Rank:                 22,
		MarketCap:            33,
		Volume24Hours:        44,
		CirculatingSupply:    55,
		Created:              statsTime,
	})
	require.NoError(t, err)

	// second interval
	statsTime = time.Date(2020, 1, 3, 14, 0, 0, 0, time.UTC)
	err = db.Insert(&models.CoinMarketCapStats{
		Price:                300,
		PercentChange24Hours: 111,
		Rank:                 222,
		MarketCap:            333,
		Volume24Hours:        444,
		CirculatingSupply:    555,
		Created:              statsTime,
	})
	require.NoError(t, err)

	// third interval
	statsTime = time.Date(2020, 1, 3, 23, 0, 0, 0, time.UTC)
	err = db.Insert(&models.CoinMarketCapStats{
		Price:                400,
		PercentChange24Hours: 1111,
		Rank:                 2222,
		MarketCap:            3333,
		Volume24Hours:        4444,
		CirculatingSupply:    5555,
		Created:              statsTime,
	})
	require.NoError(t, err)

	logger := inslogger.FromContext(context.Background())
	observerAPI := NewObserverServerFull(db, logger, pStorage, configuration.ApiExtended{
		FeeAmount:   testFee,
		Price:       testPrice,
		PriceOrigin: "coin_market_cap",
		CMCMarketStatsParams: configuration.CMCMarketStatsParamsEnabled{
			CirculatingSupply: true,
			DailyChange:       true,
			MarketCap:         true,
			Rank:              true,
			Volume:            true,
		},
	})

	e := echo.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	mockCtx := e.NewContext(req, res)

	err = observerAPI.MarketStats(mockCtx)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.Code)

	bodyBytes, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	received := ResponsesMarketStatsYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	require.Equal(t, "400", received.Price)
	require.Equal(t, "3333", *received.MarketCap)
	require.Equal(t, "2222", *received.Rank)
	require.Equal(t, "5555", *received.CirculatingSupply)
	require.Equal(t, "1111", *received.DailyChange)
	require.Equal(t, "4444", *received.Volume)

	points := *received.PriceHistory
	require.Equal(t, 3, len(points))

	require.Equal(t,
		time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
		points[0].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 8, 0, 0, 0, time.UTC).Unix(),
		points[1].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 16, 0, 0, 0, time.UTC).Unix(),
		points[2].Timestamp)

	require.Equal(t, "150", points[0].Price)
	require.Equal(t, "300", points[1].Price)
	require.Equal(t, "400", points[2].Price)
}

func TestObserverServer_Binance_Price(t *testing.T) {
	// first interval
	statsTime := time.Date(2020, 1, 3, 6, 0, 0, 0, time.UTC)
	err := db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     100,
		Symbol:             "1",
		SymbolPriceBTC:     "2",
		BTCPriceUSD:        "3",
		PriceChangePercent: "4",
		Created:            statsTime,
	})
	require.NoError(t, err)

	statsTime = time.Date(2020, 1, 3, 7, 0, 0, 0, time.UTC)
	err = db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     200,
		Symbol:             "11",
		SymbolPriceBTC:     "22",
		BTCPriceUSD:        "33",
		PriceChangePercent: "44",
		Created:            statsTime,
	})
	require.NoError(t, err)

	// second interval
	statsTime = time.Date(2020, 1, 3, 14, 0, 0, 0, time.UTC)
	err = db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     300,
		Symbol:             "111",
		SymbolPriceBTC:     "222",
		BTCPriceUSD:        "333",
		PriceChangePercent: "444",
		Created:            statsTime,
	})
	require.NoError(t, err)

	// third interval
	statsTime = time.Date(2020, 1, 3, 23, 0, 0, 0, time.UTC)
	err = db.Insert(&models.BinanceStats{
		SymbolPriceUSD:     400,
		Symbol:             "1111",
		SymbolPriceBTC:     "2222",
		BTCPriceUSD:        "3333",
		PriceChangePercent: "4444",
		Created:            statsTime,
	})
	require.NoError(t, err)

	logger := inslogger.FromContext(context.Background())
	observerAPI := NewObserverServerFull(db, logger, pStorage, configuration.ApiExtended{
		FeeAmount:   testFee,
		Price:       testPrice,
		PriceOrigin: "binance",
	})

	e := echo.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	mockCtx := e.NewContext(req, res)

	err = observerAPI.MarketStats(mockCtx)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.Code)

	bodyBytes, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	received := ResponsesMarketStatsYaml{}
	err = json.Unmarshal(bodyBytes, &received)
	require.NoError(t, err)

	require.Equal(t, "400", received.Price)
	require.Equal(t, "4444", *received.DailyChange)

	points := *received.PriceHistory
	require.Equal(t, 3, len(points))

	require.Equal(t,
		time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
		points[2].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 8, 0, 0, 0, time.UTC).Unix(),
		points[1].Timestamp)
	require.Equal(t,
		time.Date(2020, 1, 3, 16, 0, 0, 0, time.UTC).Unix(),
		points[0].Timestamp)

	require.Equal(t, "150", points[2].Price)
	require.Equal(t, "300", points[1].Price)
	require.Equal(t, "400", points[0].Price)
}
