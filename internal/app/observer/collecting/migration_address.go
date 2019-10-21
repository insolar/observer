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

package collecting

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"

	"github.com/insolar/insolar/application/builtin/contract/migrationshard"
	proxyShard "github.com/insolar/insolar/application/builtin/proxy/migrationshard"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

type MigrationAddressCollector struct {
	log     *logrus.Logger
	fetcher store.RecordFetcher
}

func NewMigrationAddressesCollector(log *logrus.Logger, fetcher store.RecordFetcher) *MigrationAddressCollector {
	return &MigrationAddressCollector{
		log:     log,
		fetcher: fetcher,
	}
}

func (c *MigrationAddressCollector) Collect(rec *observer.Record) []*observer.MigrationAddress {
	if rec == nil {
		return nil
	}

	// This code block collects addresses from incoming request.
	res := observer.CastToResult(rec)
	if res.IsResult() {
		req, err := c.fetcher.Request(context.Background(), res.Request())
		if err != nil {
			c.log.WithField("req", res.Request()).Error(errors.Wrapf(err, "result without request"))
			return nil
		}
		call, ok := c.isAddMigrationAddresses(&req)
		if !ok {
			return nil
		}

		if !res.IsSuccess() {
			// TODO: maybe we need to do something else
			c.log.Warnf("unsuccessful attempt to add migration addresses")
		}

		if call != nil {
			params := &addAddresses{}
			call.ParseMemberContractCallParams(params)
			addresses := []*observer.MigrationAddress{}
			for _, addr := range params.MigrationAddresses {
				addresses = append(addresses, &observer.MigrationAddress{
					Addr:   addr,
					Pulse:  call.ID.Pulse(),
					Wasted: false,
				})
			}
			return addresses
		}
	}

	// This code block collects addresses from genesis record.
	activate := observer.CastToActivate(rec)
	if !activate.IsActivate() {
		return nil
	}

	act := activate.Virtual.GetActivate()
	if !isMigrationShardActivate(act) {
		return nil
	}
	shard := migrationShardActivate(act)
	addresses := []*observer.MigrationAddress{}
	for _, addr := range shard {
		addresses = append(addresses, &observer.MigrationAddress{
			Addr:   addr,
			Pulse:  rec.ID.Pulse(),
			Wasted: false,
		})
	}
	return addresses
}

type addAddresses struct {
	MigrationAddresses []string `json:"migrationAddresses"`
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

func (c *MigrationAddressCollector) isAddMigrationAddresses(rec *record.Material) (*observer.Request, bool) {
	request := observer.CastToRequest((*observer.Record)(rec))
	if !request.IsIncoming() {
		return nil, false
	}

	if !request.IsMemberCall() {
		return nil, false
	}

	args := request.ParseMemberCallArguments()
	return request, args.Params.CallSite == "migration.addAddresses"
}
