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
)

type Cache interface {
	Set(insolar.ID, interface{})
	Get(insolar.ID) interface{}
	Pop(insolar.ID) interface{}
	Has(insolar.ID) bool
	Delete(insolar.ID)
}

type Predicate func(interface{}) bool
type OriginFunc func(interface{}) insolar.ID

type Chain struct {
	Parent interface{}
	Child  interface{}
}

type ChainCollector interface {
	Collect(interface{}) *Chain
}
