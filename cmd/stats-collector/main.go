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
	"flag"
	"time"

	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
)

func main() {
	collectDT := flag.String("time", "", "Historical request, format 2006-01-02 15:04:05")
	flag.Parse()

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

	var dt *time.Time
	if *collectDT != "" {
		layout := "2006-01-02 15:04:05"
		tmp, err := time.Parse(layout, *collectDT)
		if err != nil {
			log.Error("failed to parse time ", collectDT)
		}
		dt = &tmp
	}

	calcSupply(log, db, dt)
	calcNetwork(log, db)
}

func calcSupply(log *logrus.Logger, db *pg.DB, dt *time.Time) {
	repo := postgres.NewSupplyStatsRepository(db)
	sr := component.NewStatsManager(log, repo)

	command := component.NewCalculateStatsCommand(log, db, sr)
	_, err := command.Run(dt)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to run command"))
	}
}

func calcNetwork(log *logrus.Logger, db *pg.DB) {
	err := db.RunInTransaction(func(tx *pg.Tx) error {
		repo := postgres.NewNetworkStatsRepository(tx)

		stats, err := repo.CountStats()
		if err != nil {
			return errors.Wrapf(err, "failed to count network stats")
		}

		log.Debugf("Collected stats: %+v", stats)

		err = repo.InsertStats(stats)
		if err != nil {
			return errors.Wrapf(err, "failed to save network stats")
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
