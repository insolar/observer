// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package main

import (
	"context"

	echoPrometheus "github.com/globocom/echo-prometheus"
	"github.com/insolar/insconfig"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/mitchellh/mapstructure"

	insconf "github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apiconfiguration "github.com/insolar/observer/configuration/api"
	"github.com/insolar/observer/internal/app/api"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/dbconn"
)

func main() {
	cfg := &apiconfiguration.Configuration{}
	params := insconfig.Params{
		EnvPrefix:        "observerapi",
		ViperHooks:       []mapstructure.DecodeHookFunc{apiconfiguration.ToBigIntHookFunc()},
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(cfg); err != nil {
		panic(err)
	}
	insConfigurator.ToYaml(cfg)

	loggerConfig := insconf.Log{
		Level:        cfg.Log.Level,
		Formatter:    cfg.Log.Format,
		Adapter:      "zerolog",
		OutputType:   cfg.Log.OutputType,
		OutputParams: cfg.Log.OutputParams,
		BufferSize:   cfg.Log.Buffer,
	}
	_, logger := initGlobalLogger(context.Background(), loggerConfig)
	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatal(err.Error())
	}

	wa := EchoWriterAdapter{logger: logger}
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output: &wa,
	}))
	e.Use(echoPrometheus.MetricsMiddleware())
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	pStorage := postgres.NewPulseStorage(logger, db)
	observerAPI := api.NewObserverServer(db, logger, pStorage, *cfg)
	api.RegisterHandlers(e, observerAPI)

	e.Logger.Fatal(e.Start(cfg.Listen))
}

type EchoWriterAdapter struct {
	logger insolar.Logger
}

func (o *EchoWriterAdapter) Write(p []byte) (n int, err error) {
	o.logger.Info(string(p))
	return len(p), nil
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
