package observer

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
)

type Group struct {
	Ref        insolar.Reference
	Title      string
	Goal       string
	Purpose    string
	Type       string
	ChairMan   insolar.Reference
	Treasurer  insolar.Reference
	Membership foundation.StableMap
	Members    []insolar.Reference
	Status     string
	State      insolar.Reference
	Timestamp  int64
}

type GroupUpdate struct {
	PrevState      insolar.Reference
	GroupState     insolar.Reference
	GroupReference insolar.Reference
	Purpose        string
	Goal           string
	ProductType    string // TODO: create group type table
	Treasurer      insolar.Reference
	Membership     foundation.StableMap
	Timestamp      int64
}

type GroupStorage interface {
	Insert(Group) error
	Update(GroupUpdate) error
}

type GroupCollector interface {
	Collect(*Record) *Group
}
