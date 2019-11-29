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
	"context"
	"flag"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/pkg/errors"

	"github.com/insolar/observer/component"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/dbconn"
)

func main() {
	collectDT := flag.String("time", "", "Historical request, format 2006-01-02 15:04:05")
	flag.Parse()

	ctx := context.Background()
	log := inslogger.FromContext(ctx)
	cfg := configuration.Load(ctx)
	db, err := dbconn.Connect(cfg.DB)
	if err != nil {
		log.Fatal(err.Error())
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
