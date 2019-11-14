package main

import (
	"flag"

	"github.com/go-pg/migrations"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/dbconn"
)

var migrationDir = flag.String("dir", "", "directory with migrations")
var doInit = flag.Bool("init", false, "perform db init (for empty db)")

func main() {
	flag.Parse()
	cfg := configuration.Load()
	log := logrus.New()

	db := dbconn.Connect(cfg.DB)

	migrationCollection := migrations.NewCollection()
	if *doInit {
		_, _, err := migrationCollection.Run(db, "init")
		if err != nil {
			log.Fatal(errors.Wrap(err, "Could not init migrations"))
		}
	}

	err := migrationCollection.DiscoverSQLMigrations(*migrationDir)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Failed to read migrations"))
	}

	_, _, err = migrationCollection.Run(db, "up")
	if err != nil {
		log.Fatal(errors.Wrap(err, "Could not migrate"))
	}
	log.Info("migrated successfully!")
}
