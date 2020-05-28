// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package observer

import (
	"github.com/insolar/insolar/insolar"
)

type MigrationAddress struct {
	Addr   string
	Pulse  insolar.PulseNumber
	Wasted bool
}

type MigrationAddressCollector interface {
	Collect(*Record) []*MigrationAddress
}

type Vesting struct {
	Addr string
}

type VestingCollector interface {
	Collect(*Record) []*Vesting
}
