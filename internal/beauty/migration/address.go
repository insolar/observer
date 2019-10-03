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
	"context"
	"strings"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/migrationshard"
	proxyShard "github.com/insolar/insolar/logicrunner/builtin/proxy/migrationshard"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/dto"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replicator"
)

type Composer struct {
	requests map[insolar.ID]*record.Material
	results  map[insolar.ID]*record.Material
	cache    []*beauty.MigrationAddress
}

func NewComposer() *Composer {
	return &Composer{
		requests: make(map[insolar.ID]*record.Material),
		results:  make(map[insolar.ID]*record.Material),
	}
}

func (c *Composer) Init(ctx context.Context, db *pg.DB) {
	count, err := db.Model(&beauty.MigrationAddress{}).
		Where("wasted != TRUE OR wasted IS NULL").
		Count()
	if err != nil {
		return
	}
	stats.Record(ctx, migrationAddresses.M(int64(count)))
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
	case *record.Virtual_Activate:
		act := rec.Virtual.GetActivate()
		if isMigrationShardActivate(act) {
			t, err := rec.ID.Pulse().AsApproximateTime()
			if err != nil {
				return
			}
			c.processShardActivate(t, act)
		}
	}
}

func (c *Composer) Dump(
	ctx context.Context,
	tx orm.DB,
	pub replicator.OnDumpSuccess,
) error {
	log.Info("dump migration addresses")

	stats.Record(
		ctx,
		migrationAddressCache.M(int64(len(c.requests)+len(c.results))),
	)

	log.Infof("dump %d addresses", len(c.cache))
	for _, addr := range c.cache {
		if err := addr.Dump(ctx, tx); err != nil {
			return errors.Wrapf(err, "failed to dump migration addresses addr=%s", addr.Addr)
		}
	}

	pub.Subscribe(func() {
		stats.Record(ctx, migrationAddresses.M(int64(len(c.cache))))
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
	pn := req.ID.Pulse()
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

func (c *Composer) processShardActivate(t time.Time, act *record.Activate) {
	addresses := migrationShardActivate(act)
	for _, addr := range addresses {
		c.cache = append(c.cache, &beauty.MigrationAddress{
			Addr:      strings.ToLower(addr),
			Timestamp: t.Unix(),
			Wasted:    false,
		})
	}
}

func isMigrationShardActivate(act *record.Activate) bool {
	return act.Image.Equal(*proxyShard.PrototypeReference)
}

func migrationShardActivate(act *record.Activate) []string {
	if act.Memory == nil {
		return []string{}
	}
	shard := &migrationshard.MigrationShard{}
	err := insolar.Deserialize(act.Memory, shard)
	if err != nil {
		return []string{}
	}
	return shard.FreeMigrationAddresses
}
