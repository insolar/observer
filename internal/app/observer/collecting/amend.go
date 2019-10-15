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
	"github.com/insolar/observer/internal/app/observer"
)

type AmendCollector struct {
	collector observer.ChainCollector
}

func NewAmendCollector(properRequest, properAmend observer.Predicate) *AmendCollector {
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
			amend := observer.CastToAmend(chain)
			return amend.IsAmend()
		},
		Origin: func(chain interface{}) insolar.ID {
			amend := observer.CastToAmend(chain)
			return amend.Request()
		},
		Proper: properAmend,
	}
	return &AmendCollector{
		collector: NewChainCollector(parent, child),
	}
}

func (c *AmendCollector) Collect(record *observer.Record) *observer.CoupledAmend {
	request := observer.CastToRequest(record)
	amend := observer.CastToAmend(record)
	if !request.IsIncoming() && !amend.IsAmend() {
		return nil
	}

	chain := c.collector.Collect(record)
	if chain == nil {
		return nil
	}
	return &observer.CoupledAmend{
		Request: observer.CastToRequest(chain.Parent),
		Amend:   observer.CastToAmend(chain.Child),
	}
}
