package observer

type TxRegister struct {
	TransactionID        []byte
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
