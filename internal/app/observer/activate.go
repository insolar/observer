package observer

import (
	"reflect"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

type Activate record.Material

func CastToActivate(r interface{}, logger insolar.Logger) *Activate {
	if act, ok := r.(*Activate); ok {
		return act
	}
	rec, ok := r.(*Record)
	if !ok {
		logger.Warnf("trying to cast %s as *observer.Record", reflect.TypeOf(r))
		return nil
	}
	return (*Activate)(rec)
}

func (a *Activate) IsActivate() bool {
	if a == nil {
		return false
	}

	_, ok := a.Virtual.Union.(*record.Virtual_Activate)
	return ok
}

func (a *Activate) Request() insolar.ID {
	if !a.IsActivate() {
		return insolar.ID{}
	}
	result := a.Virtual.GetActivate()
	id := result.Request.GetLocal()
	if id == nil {
		return insolar.ID{}
	}
	return *id
}

type CoupledActivate struct {
	Request  *Request
	Activate *Activate
}

type ActivateCollector interface {
	Collect(*Record) *CoupledActivate
}
