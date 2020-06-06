// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package api

import (
	"fmt"
	"net/http"
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
)

type ObserverServerExtended struct {
	db       *pg.DB
	log      insolar.Logger
	pStorage observer.PulseStorage
	config   configuration.APIConfig
	server   *ObserverServer
}

func NewObserverServerExtended(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage, config configuration.APIConfig) *ObserverServerExtended {
	observerServer := NewObserverServer(db, log, pStorage, config)
	return &ObserverServerExtended{db: db, log: log, pStorage: pStorage, config: config, server: observerServer}
}

func (s *ObserverServerExtended) Fee(ctx echo.Context, amount string) error {
	s.setExpire(ctx, 1*time.Minute)

	if !isInt(amount) {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("invalid amount"))
	}
	if strings.HasPrefix(amount, "-") {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("negative amount"))
	}

	return ctx.JSON(http.StatusOK, ResponsesFeeYaml{Fee: s.config.GetFeeAmount().String()})
}

func (s *ObserverServerExtended) Notification(ctx echo.Context) error {
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

func (s *ObserverServerExtended) SupplyStatsTotal(ctx echo.Context) error {
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

func (s *ObserverServerExtended) PulseNumber(ctx echo.Context) error {
	return s.server.PulseNumber(ctx)
}

func (s *ObserverServerExtended) PulseRange(ctx echo.Context, params PulseRangeParams) error {
	return s.server.PulseRange(ctx, params)
}

// CloseTransactions returns a list of closed transactions (only with statuses `received` and `failed`).
func (s *ObserverServerExtended) ClosedTransactions(ctx echo.Context, params ClosedTransactionsParams) error {
	return s.server.ClosedTransactions(ctx, params)
}

func (s *ObserverServerExtended) Member(ctx echo.Context, reference string) error {
	return s.server.Member(ctx, reference)
}

func (s *ObserverServerExtended) MemberByPublicKey(ctx echo.Context, params MemberByPublicKeyParams) error {
	return s.server.MemberByPublicKey(ctx, params)
}

func (s *ObserverServerExtended) Balance(ctx echo.Context, reference string) error {
	return s.server.Balance(ctx, reference)
}

func (s *ObserverServerExtended) MemberTransactions(ctx echo.Context, reference string, params MemberTransactionsParams) error {
	return s.server.MemberTransactions(ctx, reference, params)
}

func (s *ObserverServerExtended) TransactionsByPulseNumberRange(ctx echo.Context, params TransactionsByPulseNumberRangeParams) error {
	return s.server.TransactionsByPulseNumberRange(ctx, params)
}

func (s *ObserverServerExtended) TransactionsSearch(ctx echo.Context, params TransactionsSearchParams) error {
	return s.server.TransactionsSearch(ctx, params)
}

func (s *ObserverServerExtended) Transaction(ctx echo.Context, txIDStr string) error {
	return s.server.Transaction(ctx, txIDStr)
}

func (s *ObserverServerExtended) setExpire(ctx echo.Context, duration time.Duration) {
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
