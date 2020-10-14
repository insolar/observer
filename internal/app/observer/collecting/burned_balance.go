// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package collecting

import (
	"errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	// "github.com/insolar/mainnet/application/builtin/contract/burned"
	// proxyBurned "github.com/insolar/mainnet/application/builtin/proxy/burned"

	"github.com/insolar/observer/internal/app/observer"
)

// import from mainnet
type BurnAccount struct {
	Balance string
}

// import from mainnet
var BurnAccountPrototypeReference = gen.Reference()

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

	var prevState insolar.ID
	var image insolar.Reference
	var memory []byte
	balance := ""
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
		return nil
	}

	if !image.Equal(BurnAccountPrototypeReference) {
		// if !image.Equal(*proxyBurned.PrototypeReference) {
		return nil
	}

	if memory == nil {
		log.Warn(errors.New("burn balance memory is nil"))
		return &observer.BurnedBalance{
			PrevState:    prevState,
			IsActivate:   isActivate,
			AccountState: rec.ID,
			Balance:      "0",
		}
	}

	b := BurnAccount{}
	// b := burned.Burned{}
	if err := insolar.Deserialize(memory, &b); err != nil {
		log.Error(errors.New("failed to deserialize burn balance memory"))
	} else {
		balance = b.Balance
	}

	return &observer.BurnedBalance{
		PrevState:    prevState,
		IsActivate:   isActivate,
		AccountState: rec.ID,
		Balance:      balance,
	}
}
