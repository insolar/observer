//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package observer

import (
	"reflect"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	log "github.com/sirupsen/logrus"
)

type Activate record.Material

func CastToActivate(r interface{}) *Activate {
	rec, ok := r.(*Record)
	if !ok {
		log.Warnf("trying to cast %s as *observer.Record", reflect.TypeOf(r))
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
