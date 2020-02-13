// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package observer

import (
	"reflect"

	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
)

type Deactivate record.Material

func CastToDeactivate(r interface{}) *Deactivate {
	if deact, ok := r.(*Deactivate); ok {
		return deact
	}
	rec, ok := r.(*Record)
	if !ok {
		log.Warnf("trying to cast %s as *observer.Record", reflect.TypeOf(r))
	}
	return (*Deactivate)(rec)
}

func (a *Deactivate) IsDeactivate() bool {
	if a == nil {
		return false
	}

	_, ok := a.Virtual.Union.(*record.Virtual_Deactivate)
	return ok
}
