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

package collecting

import (
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/v2/internal/app/observer"
)

type ActivateCollector struct {
	collector observer.ChainCollector
}

func NewActivateCollector(properRequest, properActivate observer.Predicate) *ActivateCollector {
	parent := &RelationDesc{
		Is: func(chain interface{}) bool {
			request := observer.CastToRequest(chain)
			return request.IsIncoming()
		},
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: properRequest,
	}
	child := &RelationDesc{
		Is: func(chain interface{}) bool {
			activate := observer.CastToActivate(chain)
			return activate.IsActivate()
		},
		Origin: func(chain interface{}) insolar.ID {
			activate := observer.CastToActivate(chain)
			return activate.Request()
		},
		Proper: properActivate,
	}
	return &ActivateCollector{
		collector: NewChainCollector(parent, child),
	}
}

func (c *ActivateCollector) Collect(record *observer.Record) *observer.CoupledActivate {
	request := observer.CastToRequest(record)
	activate := observer.CastToActivate(record)
	if !request.IsIncoming() && !activate.IsActivate() {
		return nil
	}

	chain := c.collector.Collect(record)
	if chain == nil {
		return nil
	}
	return &observer.CoupledActivate{
		Request:  observer.CastToRequest(chain.Parent),
		Activate: observer.CastToActivate(chain.Child),
	}
}
