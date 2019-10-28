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
		if errors.Cause(err) == store.ErrNotFound {
			c.log.Error(errors.Wrap(err, "result without request"))
			return nil
		}
		panic(errors.Wrap(err, "failed to fetch request"))
	}

	_, ok := c.isDepositCall(&req)
	if !ok {
		return nil
	}

	callTree, err := c.builder.Build(ctx, req.ID)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			c.log.Error(errors.Wrap(err, "couldn't build tree"))
			return nil
		}
		panic(errors.Wrap(err, "failed to build tree"))
	}

	var (
		activate   *record.Activate
		activateID insolar.ID
	)

	daemonCall, err := c.find(callTree.Outgoings, c.isDepositMigrationCall)
	if err != nil {
		// TODO: maybe should create failed deposit
		return nil
	}

	newCall, err := c.find(daemonCall.Outgoings, c.isDepositNew)
	if err != nil {
		// TODO: maybe should create failed deposit
		return nil
	}

	if newCall != nil {
		activateID = newCall.SideEffect.ID
		activate = newCall.SideEffect.Activation
	}

	if activate == nil {
		c.log.Warn("failed to find activation")
		return nil
	}

	d, err := c.build(activateID, activate, res)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build member"))
		return nil
	}
	return []*observer.Deposit{d}
}

func (c *DepositCollector) processGenesisRecord(ctx context.Context, rec *observer.Record) []*observer.Deposit {
	var (
		memberState      *member.Member
		walletState      *wallet.Wallet
		depositState     *deposit.Deposit
		depositRefString string
		depositID        insolar.ID
		ethHash          string
		amount           string
		balance          string
	)
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
		memberState = c.initialMemberState(activate)
		walletActivate, err := c.fetcher.SideEffect(ctx, *memberState.Wallet.GetLocal())
		if err != nil {
			c.log.WithField("wallet_ref", memberState.Wallet).
				Warnf("failed to find wallet activate record")
			continue
		}
		activate = walletActivate.Virtual.GetActivate()
		walletState = c.initialWalletState(activate)

		for _, value := range walletState.Deposits {
			depositRefString = value
			break
		}
		depositRef, err := insolar.NewReferenceFromString(depositRefString)
		if err != nil {
			c.log.WithField("deposit_ref_str", depositRefString).
				Warnf("failed to build reference from string")
			continue
		}
		if depositRef != nil {
			depositActivate, err := c.fetcher.SideEffect(ctx, *depositRef.GetLocal())
			if err != nil {
				c.log.WithField("deposit_ref", depositRef).
					Error("failed to find deposit activate record")
				continue
			}
			depositID = depositActivate.ID
			activate = depositActivate.Virtual.GetActivate()
			depositState = c.initialDepositState(activate)
			ethHash = strings.ToLower(depositState.TxHash)
			amount = depositState.Amount
			balance = depositState.Balance
		}

		timeActivate, err := rec.ID.Pulse().AsApproximateTime()
		if err != nil {
			c.log.Errorf("wrong timestamp in genesis deposit record: %+v", rec)
			continue
		}
		deposits = append(deposits, &observer.Deposit{
			EthHash:         ethHash,
			Ref:             depositID,
			Member:          *memberRef.GetLocal(),
			Timestamp:       timeActivate.Unix(),
			HoldReleaseDate: 0,
			Amount:          amount,
			Balance:         balance,
			DepositState:    depositID,
		})
	}
	return deposits
}

func (c *DepositCollector) isDepositCall(rec *record.Material) (*observer.Request, bool) {

	request := observer.CastToRequest((*observer.Record)(rec))

	if !request.IsIncoming() || !request.IsMemberCall() {
		return nil, false
	}

	args := request.ParseMemberCallArguments()
	return request, args.Params.CallSite == CallSite
}

func (c *DepositCollector) build(id insolar.ID, activate *record.Activate, res *observer.Result) (*observer.Deposit, error) {
	callResult := &migrationdaemon.DepositMigrationResult{}
	res.ParseFirstPayloadValue(callResult)
	if !res.IsSuccess() {
		return nil, errors.New("invalid create deposit result payload")
	}
	state := c.initialDepositState(activate)
	transferDate, err := id.Pulse().AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deposit create pulse (%d) to time", id.Pulse())
	}

	memberRef, err := insolar.NewIDFromString(callResult.Reference)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make memberRef from base58 string")
	}

	return &observer.Deposit{
		EthHash:         strings.ToLower(state.TxHash),
		Ref:             *activate.Request.GetLocal(),
		Member:          *memberRef,
		Timestamp:       transferDate.Unix(),
		HoldReleaseDate: 0,
		Amount:          state.Amount,
		Balance:         state.Balance,
		DepositState:    id,
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

func (c *DepositCollector) isDepositNew(req *record.IncomingRequest) bool {
	if req.Method != "New" {
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
