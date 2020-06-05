// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/models"
)

type ObserverServerFull struct {
	db       *pg.DB
	log      insolar.Logger
	pStorage observer.PulseStorage
	config   configuration.APIConfig
	server   *ObserverServer
}

func NewObserverServerFull(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage, config configuration.APIConfig) *ObserverServerFull {
	observerServer := NewObserverServer(db, log, pStorage, config)
	return &ObserverServerFull{db: db, log: log, pStorage: pStorage, config: config, server: observerServer}
}

func NewServer(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage, config configuration.APIConfig) ServerInterface {
	return NewObserverServerFull(db, log, pStorage, config)
}

func (s *ObserverServerFull) IsMigrationAddress(ctx echo.Context, ethereumAddress string) error {
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

func (s *ObserverServerFull) GetMigrationAddresses(ctx echo.Context, params GetMigrationAddressesParams) error {
	limit := params.Limit
	if limit <= 0 || limit > 1000 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("`limit` should be in range [1, 1000]"))
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
func (s *ObserverServerFull) GetMigrationAddressCount(ctx echo.Context) error {
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

func (s *ObserverServerFull) TransactionsDetails(ctx echo.Context, txID string) error {
	panic("implement me")
}

func (s *ObserverServerFull) Fee(ctx echo.Context, amount string) error {
	s.setExpire(ctx, 1*time.Minute)

	if !isInt(amount) {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("invalid amount"))
	}
	if strings.HasPrefix(amount, "-") {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("negative amount"))
	}

	return ctx.JSON(http.StatusOK, ResponsesFeeYaml{Fee: s.config.GetFeeAmount().String()})
}

func (s *ObserverServerFull) Notification(ctx echo.Context) error {
	s.setExpire(ctx, 1*time.Minute)

	res, err := component.GetNotification(ctx.Request().Context(), s.db)
	if err != nil {
		if err == component.ErrNotificationNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		s.log.Error(err)
		s.setExpire(ctx, 1*time.Second)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, ResponsesNotificationInfoYaml{
		Notification: res.Message,
	})
}

func (s *ObserverServerFull) SupplyStatsTotal(ctx echo.Context) error {
	s.setExpire(ctx, 10*time.Second)

	repo := postgres.NewSupplyStatsRepository(s.db)
	result, err := repo.LastStats()
	if err != nil && err == postgres.ErrNoStats {
		return ctx.JSON(http.StatusNoContent, "")
	}
	if err != nil {
		s.log.Error(errors.Wrap(err, "couldn't get last supply stats"))
		return ctx.JSON(http.StatusInternalServerError, "")
	}

	return ctx.String(http.StatusOK, result.TotalInXNS())
}

// PointsCount holds count of history points. Max count is 21
// https://insolar.atlassian.net/browse/INS-4049
const PointsCount = 21

func (s *ObserverServerFull) MarketStats(ctx echo.Context) error {
	s.setExpire(ctx, 1*time.Hour)
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
			DailyChange: NullableString(stats.PriceChangePercent),
		}
		response.addHistoryPoints(history)

		return ctx.JSON(http.StatusOK, response)
	case "coin_market_cap":
		checkEnabled := func(enabled bool, value float64) *string {
			if enabled {
				return NullableString(fmt.Sprintf("%v", value))
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
			CirculatingSupply: checkEnabled(s.config.GetCMCMarketStatsParams().CirculatingSupply, stats.CirculatingSupply),
			DailyChange:       checkEnabled(s.config.GetCMCMarketStatsParams().DailyChange, stats.PercentChange24Hours),
			MarketCap:         checkEnabled(s.config.GetCMCMarketStatsParams().MarketCap, stats.MarketCap),
			Price:             fmt.Sprintf("%v", stats.Price),
			Rank:              checkEnabled(s.config.GetCMCMarketStatsParams().Rank, float64(stats.Rank)),
			Volume:            checkEnabled(s.config.GetCMCMarketStatsParams().Volume, stats.Volume24Hours),
		}
		response.addHistoryPoints(history)
		return ctx.JSON(http.StatusOK, response)
	default:
		return ctx.JSON(http.StatusOK, ResponsesMarketStatsYaml{
			Price: s.config.GetPrice(),
		})
	}
}

func (s *ObserverServerFull) NetworkStats(ctx echo.Context) error {
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

func (s *ObserverServerFull) PulseNumber(ctx echo.Context) error {
	return s.server.PulseNumber(ctx)
}

func (s *ObserverServerFull) PulseRange(ctx echo.Context, params PulseRangeParams) error {
	return s.server.PulseRange(ctx, params)
}

// CloseTransactions returns a list of closed transactions (only with statuses `received` and `failed`).
func (s *ObserverServerFull) ClosedTransactions(ctx echo.Context, params ClosedTransactionsParams) error {
	return s.server.ClosedTransactions(ctx, params)
}

func (s *ObserverServerFull) Member(ctx echo.Context, reference string) error {
	return s.server.Member(ctx, reference)
}

func (s *ObserverServerFull) MemberByPublicKey(ctx echo.Context, params MemberByPublicKeyParams) error {
	return s.server.MemberByPublicKey(ctx, params)
}

func (s *ObserverServerFull) Balance(ctx echo.Context, reference string) error {
	return s.server.Balance(ctx, reference)
}

func (s *ObserverServerFull) MemberTransactions(ctx echo.Context, reference string, params MemberTransactionsParams) error {
	return s.server.MemberTransactions(ctx, reference, params)
}

func (s *ObserverServerFull) TransactionsByPulseNumberRange(ctx echo.Context, params TransactionsByPulseNumberRangeParams) error {
	return s.server.TransactionsByPulseNumberRange(ctx, params)
}

func (s *ObserverServerFull) TransactionsSearch(ctx echo.Context, params TransactionsSearchParams) error {
	return s.server.TransactionsSearch(ctx, params)
}

func (s *ObserverServerFull) Transaction(ctx echo.Context, txIDStr string) error {
	return s.server.Transaction(ctx, txIDStr)
}

func (s *ObserverServerFull) setExpire(ctx echo.Context, duration time.Duration) {
	ctx.Response().Header().Set(
		"Cache-Control",
		fmt.Sprintf("max-age=%d", int(duration.Seconds())),
	)
	ctx.Response().Header().Set(
		"Expires",
		time.Now().UTC().Add(duration).Format(http.TimeFormat),
	)
}

func isInt(s string) bool {
	s = strings.TrimPrefix(s, "-")
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
