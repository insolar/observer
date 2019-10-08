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
	ChairMan   insolar.Reference
	Membership foundation.StableMap
	Members    []insolar.Reference
	Status     string
}

type GroupStorage interface {
	Insert(Group) error
}

type GroupCollector interface {
	Collect(*Record) *Group
}
