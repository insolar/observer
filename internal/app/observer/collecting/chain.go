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
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/collecting/cache"
)

type ChainCollector struct {
	parents observer.Cache
	parent  *RelationDesc
	child   *RelationDesc
}

type RelationDesc struct {
	Is     observer.Predicate
	Origin observer.OriginFunc
	Proper observer.Predicate
}

func NewChainCollector(
	parent *RelationDesc, child *RelationDesc,
) *ChainCollector {
	return &ChainCollector{
		parents: cache.New(),
		parent:  parent,
		child:   child,
	}
}

func (c *ChainCollector) Collect(chain interface{}) *observer.Chain {
	if chain == nil {
		return nil
	}

	switch {
	case c.parent.Is(chain):
		if !c.parent.Proper(chain) {
			return nil
		}

		origin := c.origin(chain)
		if c.parents.Has(origin) {
			logrus.Warn("already has parent record, overriding")
		}
		c.parents.Set(origin, chain)
		return nil
	case c.child.Is(chain):
		origin := c.origin(chain)

		parent := c.parents.Pop(origin)
		if parent == nil {
			return nil
		}

		if !c.child.Proper(chain) {
			return nil
		}

		return &observer.Chain{
			Parent: parent,
			Child:  chain,
		}
	default:
		return nil
	}
}

func (c *ChainCollector) origin(chain interface{}) insolar.ID {
	switch {
	case c.parent.Is(chain):
		return c.parent.Origin(chain)
	case c.child.Is(chain):
		return c.child.Origin(chain)
	}
	return insolar.ID{}
}
