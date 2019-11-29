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

	echoPrometheus "github.com/globocom/echo-prometheus"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apiconfiguration "github.com/insolar/observer/configuration/api"
	"github.com/insolar/observer/internal/app/api"
	"github.com/insolar/observer/internal/dbconn"
)

func main() {
	ctx := context.Background()
	logger := inslogger.FromContext(ctx)
	cfg := apiconfiguration.Load(ctx)
	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatal(err.Error())
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(echoPrometheus.MetricsMiddleware())
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	observerAPI := api.NewObserverServer(db, logger, cfg.FeeAmount, &api.DefaultClock{}, cfg.Price)
	api.RegisterHandlers(e, observerAPI)

	e.Logger.Fatal(e.Start(cfg.Listen))
}
