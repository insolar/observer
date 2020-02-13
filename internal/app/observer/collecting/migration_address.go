// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"context"
	"fmt"

	"github.com/insolar/insolar/application/builtin/contract/migrationshard"
	proxyShard "github.com/insolar/insolar/application/builtin/proxy/migrationshard"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

type MigrationAddressCollector struct {
	log     insolar.Logger
	fetcher store.RecordFetcher
}

func NewMigrationAddressesCollector(log insolar.Logger, fetcher store.RecordFetcher) *MigrationAddressCollector {
	return &MigrationAddressCollector{
		log:     log,
		fetcher: fetcher,
	}
}

func (c *MigrationAddressCollector) Collect(ctx context.Context, rec *observer.Record) []*observer.MigrationAddress {
	if rec == nil {
		return nil
	}
	log := c.log.WithField("recordID", rec.ID.String()).WithField("collector", "MigrationAddressCollector")

	// This code block collects addresses from incoming request.
	res, err := observer.CastToResult(rec)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	if res.IsResult() {
		return c.collectFromResult(ctx, res, log)
	}

	// This code block collects addresses from genesis record.
	activate := observer.CastToActivate(rec, log)
	if activate.IsActivate() {
		return c.collectFromGenesis(ctx, rec, activate)
	}

	return nil
}

func (c *MigrationAddressCollector) collectFromResult(ctx context.Context, res *observer.Result, log insolar.Logger) []*observer.MigrationAddress {
	if !res.IsSuccess(log) {
		return nil
	}

	req, err := c.fetcher.Request(ctx, res.Request())
	if err != nil {
		panic(fmt.Sprintf("recordID %s: failed to fetch request for result", res.ID))
	}

	call, ok := c.isAddMigrationAddresses(&req, log)
	if !ok {
		return nil
	}
	if call == nil {
		return nil
	}

	params := &addAddresses{}
	call.ParseMemberContractCallParams(params, log)
	addresses := make([]*observer.MigrationAddress, 0, len(params.MigrationAddresses))
	for _, addr := range params.MigrationAddresses {
		addresses = append(addresses, &observer.MigrationAddress{
			Addr:   addr,
			Pulse:  call.ID.Pulse(),
			Wasted: false,
		})
	}
	return addresses
}

func (c *MigrationAddressCollector) collectFromGenesis(_ context.Context, rec *observer.Record, activate *observer.Activate) []*observer.MigrationAddress {
	act := activate.Virtual.GetActivate()
	if !isMigrationShardActivate(act) {
		return nil
	}
	shard := migrationShardActivate(act)
	addresses := make([]*observer.MigrationAddress, 0, len(shard))
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

func (c *MigrationAddressCollector) isAddMigrationAddresses(rec *record.Material, logger insolar.Logger) (*observer.Request, bool) {
	request := observer.CastToRequest((*observer.Record)(rec), logger)
	if !request.IsIncoming() {
		return nil, false
	}

	if !request.IsMemberCall(logger) {
		return nil, false
	}

	args := request.ParseMemberCallArguments(logger)
	return request, args.Params.CallSite == "migration.addAddresses"
}
