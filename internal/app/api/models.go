package api

type transactionType string

const (
	TTypeTransfer  transactionType = "transfer"
	TTypeMigration                 = "migration"
	TTypeVesting                   = "vesting"
)

type transactionStatus string

const (
	TStatusRegistered transactionStatus = "registered"
	TStatusSent                         = "sent"
	TStatusReceived                     = "received"
)

type member struct {
	Reference        []byte `sql:"member_ref"`
	WalletReference  []byte `sql:"wallet_ref"`
	AccountReference []byte `sql:"account_ref"`
	AccountState     []byte `sql:"account_state"`
	MigrationAddress string `sql:"migration_address"`
	Balance          string `sql:"balance"`
	Status           string `sql:"status"`
}

type deposit struct {
	Reference       []byte `sql:"deposit_ref"`
	MemberReference []byte `sql:"member_ref"`
	EtheriumHash    string `sql:"eth_hash"`
	State           []byte `sql:"deposit_state"`
	HoldReleaseDate int64  `sql:"hold_release_date"`
	Amount          string `sql:"varchar"`
	Balance         string `sql:"balance"`
}

type transaction struct {
	ID                    int64  `sql:"id"`
	TransactionID         []byte `sql:"tx_id"`
	PulseNumber           int64  `sql:"pulse_number"`
	Type                  string `sql:"type"`
	Status                string `sql:"status"`
	MemberFromReference   []byte `sql:"member_from_ref"`
	MemberToReference     []byte `sql:"member_to_ref"`
	MigrationsToReference []byte `sql:"migration_to_ref"`
	VestingFromReference  []byte `sql:"vesting_from_ref"`
	Amount                string `sql:"amount"`
	Fee                   string `sql:"fee"`
}
