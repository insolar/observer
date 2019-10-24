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
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
)

type Group struct {
	Ref        insolar.Reference
	Title      string
	Goal       string
	Image      string
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
	Image          string
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
