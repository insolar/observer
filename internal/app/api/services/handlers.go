// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package services

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/api"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

type ObserverServer struct {
	db       *pg.DB
	log      insolar.Logger
	pStorage observer.PulseStorage
	config   configuration.APIConfig
}

func NewObserverServer(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage, config configuration.APIConfig) *ObserverServer {
	return &ObserverServer{db: db, log: log, pStorage: pStorage, config: config}
}

func (s *ObserverServer) IsMigrationAddress(ctx echo.Context, ethereumAddress string) error {
	s.setExpire(ctx, 1*time.Minute)

	count, err := s.db.Model(&models.MigrationAddress{}).
		Where("addr = ?", ethereumAddress).
		Count()

	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesIsMigrationAddressYaml{
		IsMigrationAddress: count > 0,
	})
}

func (s *ObserverServer) GetMigrationAddresses(ctx echo.Context, params GetMigrationAddressesParams) error {
	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, api.NewSingleMessageError("`limit` should be in range [1, 1000]"))
	}

	query := s.db.Model(&models.MigrationAddress{}).
		Where("wasted = false")
	if params.Index != nil {
		id, err := strconv.ParseInt(*params.Index, 10, 64)
		if err != nil {
			s.log.Error(err)
			return ctx.JSON(http.StatusBadRequest, struct{}{})
		}
		query = query.Where("id > ?", id)
	}
	var result []models.MigrationAddress
	err := query.Order("id").Limit(limit).Select(&result)
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	resJSON := make(ResponsesAddressesYaml, len(result))
	for i := 0; i < len(result); i++ {
		resJSON[i].Address = result[i].Addr
		resJSON[i].Index = strconv.FormatInt(result[i].ID, 10)
	}
	return ctx.JSON(http.StatusOK, resJSON)
}

// GetMigrationAddressCount returns the total number of non-assigned migration addresses
func (s *ObserverServer) GetMigrationAddressCount(ctx echo.Context) error {
	s.setExpire(ctx, 10*time.Second)

	count, err := s.db.Model(&models.MigrationAddress{}).
		Where("wasted = false").
		Count()
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesAddressCountYaml{
		Count: count,
	})
}

func (s *ObserverServer) TransactionsDetails(ctx echo.Context, txID string) error {
	panic("implement me")
}

// PointsCount holds count of history points. Max count is 21
// https://insolar.atlassian.net/browse/INS-4049
const PointsCount = 21

func (s *ObserverServer) MarketStats(ctx echo.Context) error {
	s.setExpire(ctx, 1*time.Hour)

	totalSupply, err := s.totalSupply()
	if err != nil {
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, "")
	}
	switch s.config.GetPriceOrigin() {
	case "binance":
		repo := postgres.NewBinanceStatsRepository(s.db)
		stats, err := repo.LastStats()
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get last binance supply stats"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}

		history, err := repo.PriceHistory(PointsCount)
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get info about price history"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}
		response := ResponsesMarketStatsYaml{
			Price:       fmt.Sprintf("%v", stats.SymbolPriceUSD),
			DailyChange: api.NullableString(stats.PriceChangePercent),
			TotalSupply: totalSupply,
		}
		response.addHistoryPoints(history)

		return ctx.JSON(http.StatusOK, response)
	case "coin_market_cap":
		checkEnabled := func(enabled bool, value float64) *string {
			if enabled {
				return api.NullableString(fmt.Sprintf("%v", value))
			}
			return nil
		}
		repo := postgres.NewCoinMarketCapStatsRepository(s.db)
		stats, err := repo.LastStats()
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get last coin market cap supply stats"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}
		history, err := repo.PriceHistory(PointsCount)
		if err != nil {
			s.log.Error(errors.Wrap(err, "couldn't get info about price history"))
			return ctx.JSON(http.StatusInternalServerError, "")
		}
		response := ResponsesMarketStatsYaml{
			DailyChange: checkEnabled(s.config.GetCMCMarketStatsParams().DailyChange, stats.PercentChange24Hours),
			MarketCap:   checkEnabled(s.config.GetCMCMarketStatsParams().MarketCap, stats.MarketCap),
			Price:       fmt.Sprintf("%v", stats.Price),
			Rank:        checkEnabled(s.config.GetCMCMarketStatsParams().Rank, float64(stats.Rank)),
			Volume:      checkEnabled(s.config.GetCMCMarketStatsParams().Volume, stats.Volume24Hours),
			TotalSupply: totalSupply,
		}
		response.addHistoryPoints(history)
		return ctx.JSON(http.StatusOK, response)
	default:
		return ctx.JSON(http.StatusOK, ResponsesMarketStatsYaml{
			Price:       s.config.GetPrice(),
			TotalSupply: totalSupply,
		})
	}
}

func (s *ObserverServer) totalSupply() (*string, error) {
	repo := postgres.NewSupplyStatsRepository(s.db)
	result, err := repo.LastStats()
	if err != nil && err == postgres.ErrNoStats {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get last supply stats")
	}
	totalSupply := result.TotalInXNS()
	return api.NullableString(totalSupply), nil
}

func (s *ObserverServer) NetworkStats(ctx echo.Context) error {
	s.setExpire(ctx, 10*time.Second)

	repo := postgres.NewNetworkStatsRepository(s.db)
	result, err := repo.LastStats()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, "")
	}

	return ctx.JSON(http.StatusOK, ResponsesNetworkStatsYaml{
		Accounts:              result.TotalAccounts,
		CurrentTPS:            result.CurrentTPS,
		LastMonthTransactions: result.MonthTransactions,
		MaxTPS:                result.MaxTPS,
		Nodes:                 result.Nodes,
		TotalTransactions:     result.TotalTransactions,
	})
}

func (s *ObserverServer) setExpire(ctx echo.Context, duration time.Duration) {
	ctx.Response().Header().Set(
		"Cache-Control",
		fmt.Sprintf("max-age=%d", int(duration.Seconds())),
	)
	ctx.Response().Header().Set(
		"Expires",
		time.Now().UTC().Add(duration).Format(http.TimeFormat),
	)
}
