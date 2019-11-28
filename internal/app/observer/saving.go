package observer

import (
	"github.com/insolar/insolar/insolar"
)

type NormalSaving struct {
	Reference       insolar.Reference
	StartRoundDate  int64
	NextPaymentDate int64
	NSContribute    map[insolar.Reference]int64
	State           insolar.ID
}

type SavingUpdate struct {
	PrevState       insolar.ID
	SavingState     insolar.ID
	Reference       insolar.Reference
	StartRoundDate  int64
	NextPaymentDate int64
	NSContribute    map[insolar.Reference]int64
}

type SavingStorage interface {
	Insert(NormalSaving) error
}

type SavingCollector interface {
	Collect(*Record) *NormalSaving
}
