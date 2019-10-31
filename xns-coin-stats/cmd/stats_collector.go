package main

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
	xnscoinstats "github.com/insolar/observer/xns-coin-stats"
)

func main() {
	cfg := configuration.Load()
	opt, err := pg.ParseURL(cfg.DB.URL)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to parse cfg.DB.URL"))
	}
	db := pg.Connect(opt)
	log := logrus.New()
	err = log.Level.UnmarshalText([]byte(cfg.LogLevel))
	if err != nil {
		log.SetLevel(logrus.InfoLevel)
	}

	sr := xnscoinstats.NewStatsRepository(db, log)
	stats, err := sr.CountStats()
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to get stats"))
	}

	err = sr.InsertStats(stats)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to set stats"))
	}
}
