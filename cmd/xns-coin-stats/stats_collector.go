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

	repo := xnscoinstats.NewStatsRepository(db)
	sr := xnscoinstats.NewStatsManager(log, repo)
	stats, err := sr.CountStats()
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to get stats"))
	}

	log.Debugf("Collected stats: %+v", stats)
	err = sr.InsertStats(stats)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to set stats"))
	}
}
