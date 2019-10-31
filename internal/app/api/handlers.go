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

package api

import (
	"net/http"
	"strings"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/component"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type ObserverServer struct {
	db  *pg.DB
	log *logrus.Logger
}

func NewObserverServer(db *pg.DB, log *logrus.Logger) *ObserverServer {
	return &ObserverServer{db: db, log: log}
}

func (s *ObserverServer) GetMigrationAddresses(ctx echo.Context, params GetMigrationAddressesParams) error {
	panic("implement me")
}

func (s *ObserverServer) GetMigrationAddressCount(ctx echo.Context) error {
	panic("implement me")
}

func (s *ObserverServer) GetStatistics(ctx echo.Context) error {
	panic("implement me")
}

func (s *ObserverServer) TokenGetInfo(ctx echo.Context, params TokenGetInfoParams) error {
	panic("implement me")
}

func (s *ObserverServer) TokenWeekPrice(ctx echo.Context, interval int) error {
	panic("implement me")
}

func (s *ObserverServer) TransactionsDetails(ctx echo.Context, txID string) error {
	panic("implement me")
}

func (s *ObserverServer) ClosedTransactions(ctx echo.Context, params ClosedTransactionsParams) error {
	panic("implement me")
}

func (s *ObserverServer) Fee(ctx echo.Context, amount string) error {
	panic("implement me")
}

func (s *ObserverServer) Member(ctx echo.Context, reference string) error {
	panic("implement me")
}

func (s *ObserverServer) Balance(ctx echo.Context, reference string) error {
	panic("implement me")
}

func (s *ObserverServer) MemberTransactions(ctx echo.Context, reference string, params MemberTransactionsParams) error {
	panic("implement me")
}

func (s *ObserverServer) Notification(ctx echo.Context) error {
	panic("implement me")
}

func (s *ObserverServer) Transaction(ctx echo.Context, txIDStr string) error {
	txIDStr = strings.TrimSpace(txIDStr)

	if len(txIDStr) == 0 {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("empty tx_id"))
	}

	txID, err := insolar.NewRecordReferenceFromString(txIDStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewSingleMessageError("tx_id wrong format"))
	}

	tx, err := component.GetTx(ctx.Request().Context(), s.db, txID.Bytes())
	if err != nil {
		if err == component.ErrTxNotFound {
			return ctx.JSON(http.StatusNoContent, struct{}{})
		}
		s.log.Error(err)
		return ctx.JSON(http.StatusInternalServerError, struct{}{})
	}

	return ctx.JSON(http.StatusOK, TxToAPITx(*tx))
}

func (s *ObserverServer) TransactionsSearch(ctx echo.Context, params TransactionsSearchParams) error {
	panic("implement me")
}

func (s *ObserverServer) Coins(ctx echo.Context) error {
	panic("implement me")
}

func (s *ObserverServer) CoinsCirculating(ctx echo.Context) error {
	panic("implement me")
}

func (s *ObserverServer) CoinsMax(ctx echo.Context) error {
	panic("implement me")
}

func (s *ObserverServer) CoinsTotal(ctx echo.Context) error {
	panic("implement me")
}
