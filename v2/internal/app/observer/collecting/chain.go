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
	"github.com/insolar/observer/v2/internal/app/observer/collecting/cache"
)

type ChainCollector struct {
	parents  observer.Cache
	children observer.Cache
	others   observer.Cache
	parent   *RelationDesc
	child    *RelationDesc
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
		parents:  cache.New(),
		children: cache.New(),
		others:   cache.New(),
		parent:   parent,
		child:    child,
	}
}

func (c *ChainCollector) Collect(chain interface{}) *observer.Chain {
	origin := c.origin(chain)

	if c.others.Has(origin) {
		c.others.Delete(origin)
		return nil
	}

	switch {
	case c.parent.Proper(chain):
		if !c.children.Has(origin) {
			c.parents.Set(origin, chain)
			return nil
		}

		child := c.children.Get(origin)
		c.children.Delete(origin)

		if !c.child.Proper(child) {
			return nil
		}

		return &observer.Chain{
			Parent: chain,
			Child:  child,
		}
	case c.child.Proper(chain):
		if !c.parents.Has(origin) {
			c.children.Set(origin, chain)
			return nil
		}

		parent := c.parents.Get(origin)
		c.parents.Delete(origin)

		if !c.parent.Proper(parent) {
			return nil
		}

		return &observer.Chain{
			Parent: parent,
			Child:  chain,
		}
	}

	switch {
	case c.parents.Has(origin):
		c.parents.Delete(origin)
	case c.children.Has(origin):
		c.children.Delete(origin)
	default:
		c.others.Set(origin, chain)
	}
	return nil
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
