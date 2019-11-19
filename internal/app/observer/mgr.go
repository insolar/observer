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
	SwapProcess      Swap
	Status           string
	State            insolar.ID
}

type Swap struct {
	From insolar.Reference // User who initialized swap process
	To   insolar.Reference // User who will accept request
}

type Sequence struct {
	Member   insolar.Reference
	DrawDate int64
	IsActive bool
}

type MGRUpdate struct {
	PrevState        insolar.ID
	MGRState         insolar.ID
	GroupReference   insolar.Reference
	MGRReference     insolar.Reference
	StartRoundDate   int64
	FinishRoundDate  int64
	AmountDue        string
	PaymentFrequency string
	NextPaymentTime  int64
	Timestamp        int64
	Sequence         []Sequence
	SwapProcess      Swap
}

type MGRStorage interface {
	Insert(MGR) error
	Update(MGRUpdate) error
}

type MGRCollector interface {
	Collect(*Record) *MGR
}
