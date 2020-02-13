// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package main

import (
	"context"

	"github.com/go-pg/pg"
	insconf "github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/log"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/dbconn"
)

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
	_, logger := initGlobalLogger(context.Background(), loggerConfig)

	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatal(err.Error())
	}

	calcNetwork(logger, db)
}

func calcNetwork(log insolar.Logger, db *pg.DB) {
	repo := postgres.NewNetworkStatsRepository(db)

	stats, err := repo.CountStats()
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to count network stats"))
	}

	log.Debugf("Сollected stats: %+v", stats)

	err = repo.InsertStats(stats)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to save network stats"))
	}
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
