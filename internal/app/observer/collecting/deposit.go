// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"context"
	"strings"

	"github.com/insolar/insolar/pulse"
	"github.com/insolar/mainnet/application/builtin/contract/member"
	"github.com/insolar/mainnet/application/builtin/contract/pkshard"
	"github.com/insolar/mainnet/application/builtin/contract/wallet"

	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/internal/app/observer/tree"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/mainnet/application/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/mainnet/application/builtin/proxy/deposit"
	proxyPKShard "github.com/insolar/mainnet/application/builtin/proxy/pkshard"
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

	if !res.IsResult() {
		return nil
	}

	reqMaterial, err := c.fetcher.Request(ctx, res.Request())
	if err != nil {
		panic(errors.Wrap(err, "failed to fetch request"))
	}

	req := reqMaterial.Virtual.GetIncomingRequest()
	if req == nil {
		log.Debug("not incoming request, skipping")
		return nil
	}

	if !c.isDepositNew(req) {
		log.Debug("not deposit.New call, skipping")
		return nil
	}

	newCall, err := c.builder.Build(ctx, reqMaterial.ID)
	if err != nil {
		panic(errors.Wrap(err, "failed to build tree of request with result"))
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

	d, err := c.build(activateID, newCall.RequestID.Pulse(), activate, log)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build deposit"))
		return nil
	}

	log.Debugf("New deposit ref %s, state %s, EthHash %s", d.Ref.String(),
		d.DepositState.String(), d.EthHash)

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
				IsConfirmed:     true,
			}

			log.Debugf("New deposit ref %s, state %s, member %s, EthHash %s", d.Ref.String(),
				d.DepositState.String(), d.Member.String(), d.EthHash)

			deposits = append(deposits, d)
		}
	}
	return deposits
}

func (c *DepositCollector) build(id insolar.ID, pn pulse.Number, activate *record.Activate, log insolar.Logger) (*observer.Deposit, error) {
	transferDate, err := pn.AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deposit create pulse (%d) to time", id.Pulse())
	}

	state := c.initialDepositState(activate)
	d := &observer.Deposit{
		EthHash:      strings.ToLower(state.TxHash),
		Ref:          *insolar.NewReference(*activate.Request.GetLocal()),
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
