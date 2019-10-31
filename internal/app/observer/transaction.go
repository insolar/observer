package observer

import (
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/models"
)

type TxRegister struct {
	TransactionID        []byte
	Type                 models.TransactionType
	PulseNumber          int64
	RecordNumber         int64
	MemberFromReference  []byte
	MemberToReference    []byte
	DepositToReference   []byte
	DepositFromReference []byte
	Amount               string
}

func (t *TxRegister) Validate() error {
	if len(t.TransactionID) == 0 {
		return errors.New("TransactionID should not be empty")
	}
	if len(t.Type) == 0 {
		return errors.New("Type should not be empty")
	}
	if t.Type == models.TTypeUnknown {
		return errors.New("Type should not be Unknown")
	}
	if t.PulseNumber == 0 {
		return errors.New("PulseNumber should not be zero")
	}
	if len(t.Amount) == 0 {
		return errors.New("Amount should not be empty")
	}
	return nil
}

type TxResult struct {
	TransactionID []byte
	StatusSent    bool
	Fee           string
}

type TxSagaResult struct {
	TransactionID      []byte
	FinishSuccess      bool
	FinishPulseNumber  int64
	FinishRecordNumber int64
}
