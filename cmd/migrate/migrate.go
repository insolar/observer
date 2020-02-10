package main

import (
	"context"
	"flag"

	"github.com/go-pg/migrations"
	"github.com/insolar/insconfig"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/dbconn"
)

var migrationDir = flag.String("dir", "", "directory with migrations")
var doInit = flag.Bool("init", false, "perform db init (for empty db)")

func main() {
	params := insconfig.Params{
		ConfigStruct: configuration.Configuration{},
		EnvPrefix:    "observer",
	}
	insConfigurator := insconfig.NewInsConfigurator(params, insconfig.DefaultConfigPathGetter{
		GoFlags: flag.CommandLine,
	})
	parsedConf, err := insConfigurator.Load()
	if err != nil {
		panic(err)
	}
	cfg := parsedConf.(*configuration.Configuration)
	insConfigurator.PrintConfig(cfg)

	ctx := context.Background()
	log := inslogger.FromContext(ctx)

	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		log.Fatal(err.Error())
	}
	migrationCollection := migrations.NewCollection()
	if *doInit {
		_, _, err := migrationCollection.Run(db, "init")
		if err != nil {
			log.Fatal(errors.Wrap(err, "Could not init migrations"))
		}
	}

	err = migrationCollection.DiscoverSQLMigrations(*migrationDir)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to read migrations"))
	}

	_, _, err = migrationCollection.Run(db, "up")
	if err != nil {
		log.Fatal(errors.Wrap(err, "Could not migrate"))
	}
	log.Info("migrated successfully!")
}
