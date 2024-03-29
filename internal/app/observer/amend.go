package observer

import (
	"reflect"

	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
)

type Amend record.Material

func CastToAmend(r interface{}) *Amend {
	if amd, ok := r.(*Amend); ok {
		return amd
	}
	rec, ok := r.(*Record)
	if !ok {
		log.Warnf("trying to cast %s as *observer.Record", reflect.TypeOf(r))
	}
	return (*Amend)(rec)
}

func (a *Amend) IsAmend() bool {
	if a == nil {
		return false
	}

	_, ok := a.Virtual.Union.(*record.Virtual_Amend)
	return ok
}
