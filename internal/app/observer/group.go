package observer

import "github.com/insolar/insolar/insolar"

type Group struct {
	Ref        insolar.Reference
	Title      string
	Goal       string
	Purpose    string
	ChairMan   insolar.Reference // Owner
	Membership []insolar.Reference
	Members    []insolar.Reference
	Status     string
}

type GroupStorage interface {
	Insert(Group) error
}

type GroupCollector interface {
	Collect(*Record) *Group
}
