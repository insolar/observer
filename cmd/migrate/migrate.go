package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
)

var migrationDir = flag.String("dir", "", "directory with migrations")
var doInit = flag.Bool("init", false, "perform db init (for empty db)")

func main() {
	flag.Parse()
	cfg := configuration.Load()

	opt, err := pg.ParseURL(cfg.DB.URL)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to parse cfg.DB.URL"))
	}
	db := pg.Connect(opt)
	defer db.Close()

	migrationCollection := migrations.NewCollection()
	if *doInit {
		_, _, err = migrationCollection.Run(db, "init")
		if err != nil {
			log.Panicf("Could not init migrations: %s", err)
		}
	}

	err = migrationCollection.DiscoverSQLMigrations(*migrationDir)
	if err != nil {
		log.Panicf("Failed to read migrations: %s", err)
	}

	_, _, err = migrationCollection.Run(db, "up")
	if err != nil {
		log.Panicf("Could not migrate: %s", err)
	}
	fmt.Println("migrated successfully!")
}
