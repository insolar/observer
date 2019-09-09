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

type ResultCollector struct {
	collector observer.ChainCollector
}

func NewResultCollector(properRequest, properResult observer.Predicate) *ResultCollector {
	parent := &RelationDesc{
		Is: func(chain interface{}) bool {
			request := observer.CastToRequest(chain)
			return request.IsIncoming() || request.IsOutgoing()
		},
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: properRequest,
	}
	child := &RelationDesc{
		Is: func(chain interface{}) bool {
			result := observer.CastToResult(chain)
			return result.IsResult()
		},
		Origin: func(chain interface{}) insolar.ID {
			result := observer.CastToResult(chain)
			return result.Request()
		},
		Proper: properResult,
	}
	return &ResultCollector{
		collector: NewChainCollector(parent, child),
	}
}

func (c *ResultCollector) Collect(record *observer.Record) *observer.CoupledResult {
	request := observer.CastToRequest(record)
	result := observer.CastToResult(record)
	if !request.IsIncoming() && !request.IsOutgoing() && !result.IsResult() {
		return nil
	}

	chain := c.collector.Collect(record)
	if chain == nil {
		return nil
	}
	return &observer.CoupledResult{
		Request: observer.CastToRequest(chain.Parent),
		Result:  observer.CastToResult(chain.Child),
	}
}
