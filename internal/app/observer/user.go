package observer

import "github.com/insolar/insolar/insolar"

type User struct {
	UserRef   insolar.Reference
	KYCStatus bool
	Public    string
	Status    string
	State     []byte
}

type UserKYC struct {
	PrevState insolar.Reference
	UserState insolar.Reference
	KYC       bool
	Source    string
	Timestamp int64
}

type UserStorage interface {
	Insert(User) error
	Update(UserKYC) error
}

type UserCollector interface {
	Collect(*Record) *User
}
