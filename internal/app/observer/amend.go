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
	"github.com/insolar/insolar/insolar"
	"reflect"

	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
)

type Amend record.Material

func CastToAmend(r interface{}) *Amend {
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

func (a *Amend) Request() insolar.ID {
	if !a.IsAmend() {
		return insolar.ID{}
	}
	result := a.Virtual.GetAmend()
	id := result.Request.GetLocal()
	if id == nil {
		return insolar.ID{}
	}
	return *id
}

type CoupledAmend struct {
	Request *Request
	Amend   *Amend
}

type AmendCollector interface {
	Collect(*Record) *CoupledAmend
}
