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
