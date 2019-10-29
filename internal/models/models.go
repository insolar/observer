package models

type Member struct {
	tableName struct{} `sql:"members"` //nolint: unused,structcheck

	Reference        []byte `sql:"member_ref"`
	WalletReference  []byte `sql:"wallet_ref"`
	AccountReference []byte `sql:"account_ref"`
	AccountState     []byte `sql:"account_state"`
	MigrationAddress string `sql:"migration_address"`
	Balance          string `sql:"balance"`
	Status           string `sql:"status"`
}

type Deposit struct {
	tableName struct{} `sql:"deposits"` //nolint: unused,structcheck

	Reference       []byte `sql:"deposit_ref"`
	MemberReference []byte `sql:"member_ref"`
	EtheriumHash    string `sql:"eth_hash"`
	State           []byte `sql:"deposit_state"`
	HoldReleaseDate int64  `sql:"hold_release_date"`
	Amount          string `sql:"varchar"`
	Balance         string `sql:"balance"`
}

type TransactionStatus string

const (
	TStatusUnknown  TransactionStatus = "unknown"
	TStatusPending  TransactionStatus = "pending"
	TStatusSent     TransactionStatus = "sent"
	TStatusReceived TransactionStatus = "received"
	TStatusFailed   TransactionStatus = "failed"
)

type Transaction struct {
	tableName struct{} `sql:"simple_transactions"` //nolint: unused,structcheck

	// Indexes.
	ID            int64  `sql:"id"`
	TransactionID []byte `sql:"tx_id"`

	// Request registered.
	StatusRegistered     bool     `sql:"status_registered"`
	PulseRecord          [2]int64 `sql:"pulse_record,array"`
	MemberFromReference  []byte   `sql:"member_from_ref"`
	MemberToReference    []byte   `sql:"member_to_ref"`
	DepositToReference   []byte   `sql:"deposit_to_ref"`
	DepositFromReference []byte   `sql:"deposit_from_ref"`
	Amount               string   `sql:"amount"`
	Fee                  string   `sql:"fee"`

	// Result received.
	StatusSent bool `sql:"status_sent"`

	// Saga result received.
	StatusFinished    bool     `sql:"status_finished"`
	FinishSuccess     bool     `sql:"finish_success"`
	FinishPulseRecord [2]int64 `sql:"finish_pulse_record,array"`
}

func (t *Transaction) Status() TransactionStatus {
	registered := t.StatusRegistered
	sent := t.StatusRegistered && t.StatusSent
	finished := t.StatusRegistered && t.StatusFinished

	if finished {
		if t.FinishSuccess {
			return TStatusReceived
		}
		return TStatusFailed
	}
	if sent {
		return TStatusSent
	}
	if registered {
		return TStatusPending
	}

	return TStatusUnknown
}
