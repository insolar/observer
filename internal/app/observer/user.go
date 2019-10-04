package observer

import "github.com/insolar/insolar/insolar"

type User struct {
	UserRef   insolar.Reference
	KYCStatus bool
	Status    string
	State     []byte
}

type UserKYC struct {
	PrevState insolar.ID
	UserState insolar.ID
	KYC       bool
}

type UserStorage interface {
	Insert(User) error
	Update(UserKYC) error
}

type UserCollector interface {
	Collect(*Record) *User
}
