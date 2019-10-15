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
	State      insolar.ID
	Timestamp  int64
}

type GroupUpdate struct {
	PrevState   insolar.ID
	GroupState  insolar.ID
	Purpose     string
	Goal        string
	ProductType string // TODO: create group type table
	Treasurer   insolar.Reference
}

type GroupStorage interface {
	Insert(Group) error
	Update(GroupUpdate) error
}

type GroupCollector interface {
	Collect(*Record) *Group
}
