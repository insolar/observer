package filtering

import (
	"github.com/insolar/observer/internal/app/observer"
)

type VestingFilter struct{}

func NewVestingFilter() *VestingFilter {
	return &VestingFilter{}
}

func (*VestingFilter) Filter(vestings map[string]*observer.Vesting, addresses map[string]*observer.MigrationAddress) {
	// We try to apply migration address vesting in memory.
	for key, vesting := range vestings {
		addr, ok := addresses[vesting.Addr]
		if !ok {
			continue
		}
		addr.Wasted = true
		delete(vestings, key)
	}
}
