// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package observer

import (
	"github.com/insolar/insolar/insolar"
)

type Deposit struct {
	EthHash         string
	Ref             insolar.Reference
	Member          insolar.Reference
	Timestamp       int64
	HoldReleaseDate int64
	Amount          string
	Balance         string
	DepositState    insolar.ID
	Vesting         int64
	VestingStep     int64
	DepositNumber   int64
	IsConfirmed     bool
}

type DepositMemberUpdate struct {
	Ref    insolar.Reference
	Member insolar.Reference
}

type DepositUpdate struct {
	ID              insolar.ID
	Timestamp       int64
	Lockup          int64
	HoldReleaseDate int64
	Amount          string
	Balance         string
	// Prev state record ID
	PrevState   insolar.ID
	TxHash      string // for debug purposes
	IsConfirmed bool
}
