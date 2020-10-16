// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/mainnet/application/builtin/contract/burnedaccount"
	proxyBurned "github.com/insolar/mainnet/application/builtin/proxy/burnedaccount"

	"github.com/insolar/observer/internal/app/observer"
)

type BurnedBalanceCollector struct {
	log insolar.Logger
}

func NewBurnedBalanceCollector(log insolar.Logger) *BurnedBalanceCollector {
	return &BurnedBalanceCollector{
		log: log,
	}
}

func (c *BurnedBalanceCollector) Collect(rec *observer.Record) *observer.BurnedBalance {
	if rec == nil {
		return nil
	}

	b, memory := burnedBalance(rec)
	if b == nil {
		return nil
	}
	if memory == nil {
		log.Warn(errors.New("burn balance memory is nil"))
		return b
	}

	burnAccount := burnedaccount.BurnedAccount{}
	if err := insolar.Deserialize(memory, &burnAccount); err != nil {
		log.Error(errors.New("failed to deserialize burn balance memory"))
	}
	b.Balance = burnAccount.Balance

	return b
}

func burnedBalance(rec *observer.Record) (*observer.BurnedBalance, []byte) {
	var prevState insolar.ID
	var image insolar.Reference
	var memory []byte
	isActivate := false
	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
		isActivate = true
		image = v.Activate.Image
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
		prevState = v.Amend.PrevState
		image = v.Amend.Image
	default:
		log.Error(errors.New("invalid record to get burned balance memory"))
		return nil, nil
	}

	if !image.Equal(*proxyBurned.PrototypeReference) {
		return nil, nil
	}

	return &observer.BurnedBalance{
		PrevState:    prevState,
		IsActivate:   isActivate,
		AccountState: rec.ID,
		Balance:      "0",
	}, memory

}
