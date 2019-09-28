package observer

import "github.com/insolar/insolar/insolar"

type User struct {
	UserRef   insolar.Reference
	KYCStatus bool
	Status    string
}

type UserStorage interface {
	Insert(User) error
}

type UserCollector interface {
	Collect(*Record) *User
}
