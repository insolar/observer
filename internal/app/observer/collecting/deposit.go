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
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
)

const (
	CallSite = "deposit.migration"
)

type DepositCollector struct {
	log     *logrus.Logger
	fetcher store.RecordFetcher
	builder tree.Builder
}

func NewDepositCollector(log *logrus.Logger, fetcher store.RecordFetcher) *DepositCollector {
	return &DepositCollector{
		log:     log,
		fetcher: fetcher,
		builder: tree.NewBuilder(fetcher),
	}
}

func (c *DepositCollector) Collect(ctx context.Context, rec *observer.Record) []*observer.Deposit {
	if rec == nil {
		return nil
	}

	// genesis deposit records
	if rec.ID.Pulse() == insolar.GenesisPulse.PulseNumber && isPKShardActivate(rec) {
		return c.processGenesisRecord(ctx, rec)
	}

	res := observer.CastToResult(rec)
	if !res.IsResult() || !res.IsSuccess() {
		return nil
	}

	req, err := c.fetcher.Request(ctx, res.Request())
	if err != nil {
		panic(errors.Wrap(err, "failed to fetch request"))
	}

	incReq := req.Virtual.GetIncomingRequest()
	if incReq == nil {
		return nil
	}

	ok := c.isDepositMigrationCall(incReq)
	if !ok {
		return nil
	}

	migrationTree, err := c.builder.Build(ctx, req.ID)
	if err != nil {
		panic(errors.Wrap(err, "failed to build tree"))
	}

	confirmCall, err := c.find(migrationTree.Outgoings, c.isConfirmCall)
	if err != nil {
		// no confirm, construction probably
		return nil
	}

	_, err = c.find(confirmCall.Outgoings, c.isTransferToDepositCall)
	if err != nil {
		// no transfer, not enough confirms yet
		return nil
	}

	if confirmCall.SideEffect == nil {
		return nil
	}
	if confirmCall.SideEffect.Amend == nil {
		panic(errors.Wrap(err, "confirm call has side effect, but it's not amend"))
	}

	depositStateID := confirmCall.SideEffect.ID

	depositState := deposit.Deposit{}
	err = insolar.Deserialize(confirmCall.SideEffect.Amend.Memory, &depositState)
	if err != nil {
		panic(errors.New("failed to deserialize deposit contract state"))
	}

	depositID := confirmCall.Request.Object.GetLocal()

	d, err := c.build(*depositID, depositStateID, &depositState, res)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build member"))
		return nil
	}
	return []*observer.Deposit{d}
}

func (c *DepositCollector) processGenesisRecord(ctx context.Context, rec *observer.Record) []*observer.Deposit {
	activate := rec.Virtual.GetActivate()
	shard := c.initialPKShard(activate)
	var (
		deposits []*observer.Deposit
	)
	for _, memberRefStr := range shard.Map {
		memberRef, err := insolar.NewReferenceFromString(memberRefStr)
		if err != nil {
			c.log.WithField("member_ref_str", memberRefStr).
				Errorf("failed to build reference from string")
			continue
		}
		memberActivate, err := c.fetcher.SideEffect(ctx, *memberRef.GetLocal())
		if err != nil {
			c.log.WithField("member_ref", memberRef).
				Error("failed to find member activate record")
			continue
		}
		activate := memberActivate.Virtual.GetActivate()
		memberState := c.initialMemberState(activate)
		walletActivate, err := c.fetcher.SideEffect(ctx, *memberState.Wallet.GetLocal())
		if err != nil {
			c.log.WithField("wallet_ref", memberState.Wallet).
				Warnf("failed to find wallet activate record")
			continue
		}
		activate = walletActivate.Virtual.GetActivate()
		walletState := c.initialWalletState(activate)

		for _, depositRefString := range walletState.Deposits {
			depositRef, err := insolar.NewReferenceFromString(depositRefString)
			if err != nil {
				c.log.WithField("deposit_ref_str", depositRefString).
					Warnf("failed to build reference from string")
				continue
			}

			depositActivate, err := c.fetcher.SideEffect(ctx, *depositRef.GetLocal())
			if err != nil {
				c.log.WithField("deposit_ref", depositRef).
					Error("failed to find deposit activate record")
				continue
			}

			timeActivate, err := depositRef.GetLocal().Pulse().AsApproximateTime()
			if err != nil {
				c.log.Errorf("wrong timestamp in genesis deposit record: %+v", rec)
				continue
			}

			activate = depositActivate.Virtual.GetActivate()
			depositState := c.initialDepositState(activate)

			hrd, err := depositState.PulseDepositUnHold.AsApproximateTime()
			if err != nil {
				c.log.Errorf("wrong timestamp in genesis deposit PulseDepositUnHold: %+v", depositState)
				hrd, _ = pulse.Number(pulse.MinTimePulse).AsApproximateTime()
			}
			deposits = append(deposits, &observer.Deposit{
				EthHash:         strings.ToLower(depositState.TxHash),
				Ref:             *depositRef,
				DepositState:    depositActivate.ID,
				Member:          *memberRef,
				Timestamp:       timeActivate.Unix(),
				Amount:          depositState.Amount,
				Balance:         depositState.Balance,
				HoldReleaseDate: hrd.Unix(),
				Vesting:         depositState.Vesting,
				VestingStep:     depositState.VestingStep,
			})
		}
	}
	return deposits
}

func (c *DepositCollector) build(id insolar.ID, stateID insolar.ID, state *deposit.Deposit, res *observer.Result) (*observer.Deposit, error) {
	callResult := migrationdaemon.DepositMigrationResult{}
	res.ParseFirstPayloadValue(&callResult)
	if !res.IsSuccess() {
		return nil, errors.New("invalid create deposit result payload")
	}
	transferDate, err := id.Pulse().AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deposit create pulse (%d) to time", id.Pulse())
	}

	memberRef, err := insolar.NewReferenceFromString(callResult.Reference)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make memberRef from base58 string")
	}

	hrd, err := state.PulseDepositUnHold.AsApproximateTime()
	if err != nil {
		c.log.Errorf("wrong timestamp in deposit PulseDepositUnHold: %+v", state)
		hrd, _ = pulse.Number(pulse.MinTimePulse).AsApproximateTime()
	}
	return &observer.Deposit{
		EthHash:         strings.ToLower(state.TxHash),
		Ref:             *insolar.NewReference(id),
		Member:          *memberRef,
		Timestamp:       transferDate.Unix(),
		HoldReleaseDate: hrd.Unix(),
		Amount:          state.Amount,
		Balance:         state.Balance,
		DepositState:    stateID,
		Vesting:         state.Vesting,
		VestingStep:     state.VestingStep,
	}, nil
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

func (c *DepositCollector) isConfirmCall(req *record.IncomingRequest) bool {
	if req.Method != "Confirm" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func (c *DepositCollector) isTransferToDepositCall(req *record.IncomingRequest) bool {
	if req.Method != "TransferToDeposit" {
		return false
	}

	if req.Prototype == nil {
		return false
	}

	return req.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func isPKShardActivate(rec *observer.Record) bool {
	activate := observer.CastToActivate(rec)
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
