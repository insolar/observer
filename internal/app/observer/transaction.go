package observer

import (
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
