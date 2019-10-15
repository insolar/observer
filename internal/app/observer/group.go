package observer

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
)

type Group struct {
	Ref        insolar.ID
	Title      string
	Goal       string
	Purpose    string
	Type       string
	ChairMan   insolar.ID
	Treasurer  insolar.ID
	Membership foundation.StableMap
	Members    []insolar.ID
	Status     string
	State      insolar.ID
	Timestamp  int64
}

type GroupUpdate struct {
	PrevState      insolar.ID
	GroupState     insolar.ID
	GroupReference insolar.ID
	Purpose        string
	Goal           string
	ProductType    string // TODO: create group type table
	Treasurer      insolar.ID
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
