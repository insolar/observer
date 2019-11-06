package observer

import (
	"github.com/insolar/insolar/insolar"
)

type MGR struct {
	Ref              insolar.Reference
	GroupReference   insolar.Reference
	StartRoundDate   int64
	FinishRoundDate  int64
	AmountDue        string
	PaymentFrequency string
	NextPaymentTime  int64
	Sequence         []Sequence
	Status           string
	State            insolar.Reference
}

type Sequence struct {
	Member   insolar.Reference
	DueDate  int64
	IsActive bool
}

type MGRUpdate struct {
	PrevState        insolar.Reference
	MGRState         insolar.Reference
	GroupReference   insolar.Reference
	MGRReference     insolar.Reference
	StartRoundDate   int64
	FinishRoundDate  int64
	AmountDue        string
	PaymentFrequency string
	NextPaymentTime  int64
	Sequence         []Sequence
}

type MGRStorage interface {
	Insert(MGR) error
	Update(MGRUpdate) error
}

type MGRCollector interface {
	Collect(*Record) *MGR
}
