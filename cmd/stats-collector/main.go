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
	insconf "github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/log"
	"github.com/pkg/errors"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/dbconn"
)

func main() {
	collectDT := flag.String("time", "", "Historical request, format 2006-01-02 15:04:05")
	flag.Parse()

	cfg := configuration.Load()
	logger, err := log.NewLog(insconf.Log{
		Level:      cfg.Log.Level,
		Formatter:  cfg.Log.Format,
		Adapter:    "zerolog",
		OutputType: "stderr",
		BufferSize: 0,
	})
	if err != nil {
		log.Fatalf("Can't create logger: %s", err.Error())
	}
	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		logger.Fatal(err.Error())
	}

	var dt *time.Time
	if *collectDT != "" {
		layout := "2006-01-02 15:04:05"
		tmp, err := time.Parse(layout, *collectDT)
		if err != nil {
			logger.Error("failed to parse time ", collectDT)
		}
		dt = &tmp
	}

	calcSupply(logger, db, dt)
	calcNetwork(logger, db)
}

func calcSupply(log insolar.Logger, db *pg.DB, dt *time.Time) {
	repo := postgres.NewSupplyStatsRepository(db)
	sr := component.NewStatsManager(log, repo)

	command := component.NewCalculateStatsCommand(log, db, sr)
	_, err := command.Run(dt)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to run command"))
	}
}

func calcNetwork(log insolar.Logger, db *pg.DB) {
	repo := postgres.NewNetworkStatsRepository(db)

	stats, err := repo.CountStats()
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to count network stats"))
	}

	log.Debugf("Collected stats: %+v", stats)

	err = repo.InsertStats(stats)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to save network stats"))
	}
}
