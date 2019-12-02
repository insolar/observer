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
	"strings"

	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/application/builtin/contract/pkshard"
	"github.com/insolar/insolar/application/builtin/contract/wallet"
	"github.com/insolar/insolar/pulse"

	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"

	"github.com/insolar/insolar/application/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	"github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	proxyDaemon "github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	proxyPKShard "github.com/insolar/insolar/application/builtin/proxy/pkshard"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
)

type DepositCollector struct {
	log     insolar.Logger
	fetcher store.RecordFetcher
	builder tree.Builder
}

func NewDepositCollector(log insolar.Logger, fetcher store.RecordFetcher) *DepositCollector {
	return &DepositCollector{
		log:     log,
		fetcher: fetcher,
		builder: tree.NewBuilder(fetcher),
	}
}

func (c *DepositCollector) Collect(ctx context.Context, rec *observer.Record) []observer.Deposit {
	if rec == nil {
		return nil
	}

	log := c.log.WithField("recordID", rec.ID.String()).WithField("collector", "DepositCollector")

	// genesis deposit records
	if rec.ID.Pulse() == insolar.GenesisPulse.PulseNumber && isPKShardActivate(rec, log) {
		log.Debug("found genesis deposit")
		return c.processGenesisRecord(ctx, rec, log)
	}

	res, err := observer.CastToResult(rec)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	if !res.IsResult() || !res.IsSuccess(log) {
		return nil
	}

	req, err := c.fetcher.Request(ctx, res.Request())
	if err != nil {
		panic(errors.Wrap(err, "failed to fetch request"))
	}

	if !c.isDepositMigrationAPICallSite(&req, log) {
		return nil
	}

	migrationTree, err := c.builder.Build(ctx, req.ID)
	if err != nil {
		panic(errors.Wrap(err, "failed to build tree"))
	}

	daemonCall, err := c.find(migrationTree.Outgoings, c.isDepositMigrationCall)
	if err != nil {
		log.Error("deposit.migration call site didn't result in DepositMigration call")
		return nil
	}

	newCall, err := c.find(daemonCall.Outgoings, c.isDepositNew)
	if err != nil {
		log.Debug("no deposit constructor call, probably second or third confirmation, skipping")
		return nil
	}

	var (
		activate   *record.Activate
		activateID insolar.ID
	)

	if newCall.SideEffect != nil {
		activateID = newCall.SideEffect.ID
		activate = newCall.SideEffect.Activation
	}

	if activate == nil {
		log.Error("deposit's constructor request has no activation side effect")
		return nil
	}

	d, err := c.build(activateID, newCall.RequestID.Pulse(), activate, res, log)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build member"))
		return nil
	}

	log.Debugf("New deposit ref %s, state %s, member %s, EthHash %s", d.Ref.String(),
		d.DepositState.String(), d.Member.String(), d.EthHash)

	return []observer.Deposit{*d}
}

func (c *DepositCollector) processGenesisRecord(ctx context.Context, rec *observer.Record, log insolar.Logger) []observer.Deposit {
	activate := rec.Virtual.GetActivate()
	shard := c.initialPKShard(activate)
	var (
		deposits []observer.Deposit
	)
	for _, memberRefStr := range shard.Map {
		memberRef, err := insolar.NewReferenceFromString(memberRefStr)
		if err != nil {
			log.WithField("member_ref_str", memberRefStr).
				Error("failed to build reference from string")
			continue
		}
		memberActivate, err := c.fetcher.SideEffect(ctx, *memberRef.GetLocal())
		if err != nil {
			log.WithField("member_ref", memberRef).
				Error("failed to find member activate record")
			continue
		}
		activate := memberActivate.Virtual.GetActivate()
		memberState := c.initialMemberState(activate)
		// Deposit migration members has no wallet
		if memberState.Wallet.IsEmpty() {
			log.Debug("Member has no wallet. ", memberRef)
			continue
		}
		walletActivate, err := c.fetcher.SideEffect(ctx, *memberState.Wallet.GetLocal())
		if err != nil {
			log.WithField("wallet_ref", memberState.Wallet).
				Warn("failed to find wallet activate record")
			continue
		}
		activate = walletActivate.Virtual.GetActivate()
		walletState := c.initialWalletState(activate)

		for _, depositRefString := range walletState.Deposits {
			depositRef, err := insolar.NewReferenceFromString(depositRefString)
			if err != nil {
				log.WithField("deposit_ref_str", depositRefString).
					Warn("failed to build reference from string")
				continue
			}

			depositActivate, err := c.fetcher.SideEffect(ctx, *depositRef.GetLocal())
			if err != nil {
				log.WithField("deposit_ref", depositRef).
					Error("failed to find deposit activate record")
				continue
			}

			activate = depositActivate.Virtual.GetActivate()
			depositState := c.initialDepositState(activate)

			hrd, err := depositState.PulseDepositUnHold.AsApproximateTime()
			if err != nil {
				log.Errorf("wrong timestamp in genesis deposit PulseDepositUnHold: %+v", depositState)
				hrd, _ = pulse.Number(pulse.MinTimePulse).AsApproximateTime()
			}

			d := observer.Deposit{
				EthHash:         strings.ToLower(depositState.TxHash),
				Ref:             *depositRef,
				DepositState:    depositActivate.ID,
				Member:          *memberRef,
				Amount:          depositState.Amount,
				Balance:         depositState.Balance,
				Timestamp:       hrd.Unix() - depositState.Lockup,
				HoldReleaseDate: hrd.Unix(),
				Vesting:         depositState.Vesting,
				VestingStep:     depositState.VestingStep,
			}

			log.Debugf("New deposit ref %s, state %s, member %s, EthHash %s", d.Ref.String(),
				d.DepositState.String(), d.Member.String(), d.EthHash)

			deposits = append(deposits, d)
		}
	}
	return deposits
}

func (c *DepositCollector) build(id insolar.ID, pn pulse.Number, activate *record.Activate, res *observer.Result, log insolar.Logger) (*observer.Deposit, error) {
	callResult := migrationdaemon.DepositMigrationResult{}
	res.ParseFirstPayloadValue(&callResult, log)
	if !res.IsSuccess(log) {
		return nil, errors.New("invalid create deposit result payload")
	}
	transferDate, err := pn.AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deposit create pulse (%d) to time", id.Pulse())
	}

	memberRef, err := insolar.NewReferenceFromString(callResult.Reference)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make memberRef from base58 string")
	}

	state := c.initialDepositState(activate)
	d := &observer.Deposit{
		EthHash:      strings.ToLower(state.TxHash),
		Ref:          *insolar.NewReference(*activate.Request.GetLocal()),
		Member:       *memberRef,
		Timestamp:    transferDate.Unix(),
		Amount:       state.Amount,
		Balance:      state.Balance,
		DepositState: id,
		Vesting:      state.Vesting,
		VestingStep:  state.VestingStep,
	}

	if state.PulseDepositUnHold > 0 {
		hrd, err := state.PulseDepositUnHold.AsApproximateTime()
		if err != nil {
			log.Errorf("wrong timestamp in deposit PulseDepositUnHold: %+v", state)
		} else {
			d.HoldReleaseDate = hrd.Unix()
		}
	}

	return d, nil
}

func (c *DepositCollector) find(outs []tree.Outgoing, predicate func(*record.IncomingRequest) bool) (*tree.Structure, error) {
	for _, req := range outs {
		if req.Structure == nil {
			continue
		}

		if predicate(&req.Structure.Request) {
			return req.Structure, nil
		}
	}
	return nil, errors.New("failed to find corresponding request in calls tree")
}

func (c *DepositCollector) isDepositMigrationCall(req *record.IncomingRequest) bool {
	if req.Method != "DepositMigrationCall" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDaemon.PrototypeReference)
}

func (c *DepositCollector) isDepositNew(req *record.IncomingRequest) bool {
	if req.Method != "New" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func isPKShardActivate(rec *observer.Record, logger insolar.Logger) bool {
	activate := observer.CastToActivate(rec, logger)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()
	return act.Image.Equal(*proxyPKShard.PrototypeReference)
}

func (c *DepositCollector) initialPKShard(act *record.Activate) *pkshard.PKShard {
	shard := pkshard.PKShard{}
	err := insolar.Deserialize(act.Memory, &shard)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize pkshard contract state"))
	}
	return &shard
}

func (c *DepositCollector) initialMemberState(act *record.Activate) *member.Member {
	m := member.Member{}
	err := insolar.Deserialize(act.Memory, &m)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize member contract state"))
	}
	return &m
}

func (c *DepositCollector) initialWalletState(act *record.Activate) *wallet.Wallet {
	w := wallet.Wallet{}
	err := insolar.Deserialize(act.Memory, &w)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize wallet contract state"))
	}
	return &w
}

func (c *DepositCollector) initialDepositState(act *record.Activate) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(act.Memory, &d)
	if err != nil {
		c.log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}

func (c *DepositCollector) isDepositMigrationAPICallSite(rec *record.Material, logger insolar.Logger) bool {
	request := observer.CastToRequest((*observer.Record)(rec), logger)

	if !request.IsIncoming() || !request.IsMemberCall(logger) {
		return false
	}

	args := request.ParseMemberCallArguments(logger)
	return args.Params.CallSite == "deposit.migration"
}
