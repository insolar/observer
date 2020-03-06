// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/log"

	insconf "github.com/insolar/insolar/configuration"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
)

var stop = make(chan os.Signal, 1)

func main() {
	cfg := configuration.Load()
	loggerConfig := insconf.Log{
		Level:        cfg.Log.Level,
		Formatter:    cfg.Log.Format,
		Adapter:      "zerolog",
		OutputType:   cfg.Log.OutputType,
		OutputParams: cfg.Log.OutputParams,
		BufferSize:   cfg.Log.Buffer,
	}
	ctx, logger := initGlobalLogger(context.Background(), loggerConfig)
	manager := component.Prepare(ctx, cfg)
	manager.Start()
	graceful(logger, manager.Stop)
}

func graceful(logger insolar.Logger, that func()) {
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Infof("gracefully stopping...")
	that()
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
