package observer

import (
	"context"

	"github.com/insolar/insolar/insolar"
)

// Member describes insolar member.
type Member struct {
	MemberRef        insolar.Reference
	Balance          string
	MigrationAddress string
	AccountState     insolar.ID
	Status           string
	WalletRef        insolar.Reference
	AccountRef       insolar.Reference
	PublicKey        string
}

type Balance struct {
	PrevState    insolar.ID
	AccountState insolar.ID
	Balance      string
}

type BurnedBalance struct {
	PrevState    insolar.ID
	AccountState insolar.ID
	IsActivate   bool
	Balance      string
}

type MemberCollector interface {
	Collect(context.Context, *Record) *Member
}

type BalanceCollector interface {
	Collect(*Record) *Balance
}

type MemberStorage interface {
	Insert(*Member) error
	Update(*Balance) error
}

type BalanceFilter interface {
	Filter(map[insolar.ID]*Balance, map[insolar.ID]*Member)
}
