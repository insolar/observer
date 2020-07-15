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

	"github.com/insolar/insconfig"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/log"

	insconf "github.com/insolar/insolar/configuration"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
)

var stop = make(chan os.Signal, 1)
var Version string

func main() {
	cfg := &configuration.Observer{}
	params := insconfig.Params{
		EnvPrefix:        "observer",
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
	ctx, logger := initGlobalLogger(context.Background(), loggerConfig)
	if len(Version) == 0 {
		logger.Fatal("Failed to determine the version of the Observer. please use the command `make build`or `make build-node`")
	}
	logger.Infof("Observer version=%s", Version)

	manager := component.Prepare(ctx, cfg, Version)
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
