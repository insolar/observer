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
	"github.com/labstack/echo/v4"
)

type ObserverServer struct{}

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

func (s *ObserverServer) Transaction(ctx echo.Context, txID string) error {
	panic("implement me")
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
