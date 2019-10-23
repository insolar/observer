package observer

type TxRegister struct {
	TransactionID         []byte
	PulseNumber           int64
	MemberFromReference   []byte
	MemberToReference     []byte
	MigrationsToReference []byte
	VestingFromReference  []byte
	Amount                string
	Fee                   string
}

type TxResult struct {
	TransactionID []byte
	StatusSent    bool
}

type TxSagaResult struct {
	TransactionID      []byte
	FinishSuccess      bool
	FinishPulseNumber  int64
	FinishRecordNumber int64
}
