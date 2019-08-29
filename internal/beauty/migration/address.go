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

package migration

import (
	"strings"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/pulse"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/dto"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replication"
)

type Composer struct {
	requests map[insolar.ID]*record.Material
	results  map[insolar.ID]*record.Material
	cache    []*beauty.MigrationAddress

	migrationAddressGauge prometheus.Gauge
	stat                  *dumpStat
}

func NewComposer(migrationAddressGauge prometheus.Gauge) *Composer {
	stat := &dumpStat{
		cached: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "observer_migration_address_composer_cached_total",
			Help: "Cache size of migration address composer",
		}),
	}
	return &Composer{
		requests:              make(map[insolar.ID]*record.Material),
		results:               make(map[insolar.ID]*record.Material),
		migrationAddressGauge: migrationAddressGauge,
		stat:                  stat,
	}
}

func (c *Composer) Init(db *pg.DB) {
	count, err := db.Model(&beauty.MigrationAddress{}).
		Where("wasted != TRUE OR wasted IS NULL").
		Count()
	if err != nil {
		return
	}
	c.migrationAddressGauge.Set(float64(count))
}

func (c *Composer) Process(rec *record.Material) {
	defer panic.Log("migration_address_composer")

	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Result:
		origin := *v.Result.Request.GetLocal()
		if req, ok := c.requests[origin]; ok {
			delete(c.requests, origin)
			request := (*dto.Request)(req)
			if request.IsIncoming() {
				if isAddMigrationAddresses(req) {
					c.processAddMigrationAddresses(req, rec)
				}
			}
		} else {
			c.results[origin] = rec
		}
	case *record.Virtual_IncomingRequest:
		origin := rec.ID
		if res, ok := c.results[origin]; ok {
			delete(c.results, origin)
			if isAddMigrationAddresses(rec) {
				c.processAddMigrationAddresses(rec, res)
			}
		} else {
			c.requests[origin] = rec
		}
	case *record.Virtual_OutgoingRequest:
		origin := rec.ID
		if _, ok := c.results[origin]; ok {
			delete(c.results, origin)
		} else {
			c.requests[origin] = rec
		}
	}
}

func (c *Composer) Dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	log.Infof("dump migration addresses")

	c.updateStat()

	log.Infof("dump %d addresses", len(c.cache))
	for _, addr := range c.cache {
		if err := addr.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump migration addresses addr=%s", addr.Addr)
		}
	}

	pub.Subscribe(func() {
		c.migrationAddressGauge.Add(float64(len(c.cache)))
		c.cache = []*beauty.MigrationAddress{}
	})
	return nil
}

func (c *Composer) processAddMigrationAddresses(req *record.Material, res *record.Material) {
	result := (*dto.Result)(res)
	if !result.IsSuccess() {
		return
	}
	args := (*dto.Request)(req).ParseMemberCallArguments()
	addresses := parseAddMigrationAddressesCallParams(args)
	pn := pulse.Number(req.ID.Pulse())
	t, err := pn.AsApproximateTime()
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to convert AddMigrationAddresses request pulse to time"))
	}
	for _, addr := range addresses {
		c.cache = append(c.cache, &beauty.MigrationAddress{
			Addr:      strings.ToLower(addr),
			Timestamp: t.Unix(),
			Wasted:    false,
		})
	}
}

func isAddMigrationAddresses(rec *record.Material) bool {
	request := (*dto.Request)(rec)
	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "migration.addAddresses"
}

type dumpStat struct {
	cached prometheus.Gauge
}

func (c *Composer) updateStat() {
	requestCount := len(c.requests)
	resultCount := len(c.results)

	c.stat.cached.Set(float64(requestCount + resultCount))
}
