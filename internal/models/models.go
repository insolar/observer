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

type TransactionType string

const (
	TTYpeUnknown   TransactionType = "unknown"
	TTypeTransfer  TransactionType = "transfer"
	TTypeMigration TransactionType = "migration"
	TTypeRelease   TransactionType = "release"
)

func TransactionColumns() []string {
	return []string{
		"tx_id",
		"status_registered",
		"pulse_number",
		"record_number",
		"member_from_ref",
		"member_to_ref",
		"migration_to_ref",
		"vesting_from_ref",
		"amount",
		"fee",
	}
}

type Transaction struct {
	tableName struct{} `sql:"simple_transactions"` //nolint: unused,structcheck

	// Indexes.
	ID            int64  `sql:"id"`
	TransactionID []byte `sql:"tx_id"`

	// Request registered.
	StatusRegistered      bool   `sql:"status_registered"`
	PulseNumber           int64  `sql:"pulse_number"`
	RecordNumber          int64  `sql:"record_number"`
	MemberFromReference   []byte `sql:"member_from_ref"`
	MemberToReference     []byte `sql:"member_to_ref"`
	MigrationsToReference []byte `sql:"migration_to_ref"`
	VestingFromReference  []byte `sql:"vesting_from_ref"`
	Amount                string `sql:"amount"`
	Fee                   string `sql:"fee"`

	// Result received.
	StatusSent bool `sql:"status_sent"`

	// Saga result received.
	StatusFinished     bool  `sql:"status_finished"`
	FinishSuccess      bool  `sql:"finish_success"`
	FinishPulseNumber  int64 `sql:"finish_pulse_number"`
	FinishRecordNumber int64 `sql:"finish_record_number"`
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

func (t *Transaction) Type() TransactionType {
	if len(t.VestingFromReference) > 0 {
		return TTypeRelease
	}
	if len(t.MigrationsToReference) > 0 {
		return TTypeMigration
	}

	if len(t.MemberFromReference) > 0 && len(t.MemberToReference) > 0 {
		return TTypeTransfer
	}

	return TTYpeUnknown
}
