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

package cache

import (
	"github.com/insolar/insolar/insolar"
)

type Cache struct {
	Cache map[insolar.ID]interface{}
}

func New() *Cache {
	return &Cache{
		Cache: make(map[insolar.ID]interface{}),
	}
}

func (c *Cache) Set(key insolar.ID, value interface{}) {
	c.Cache[key] = value
}

func (c *Cache) Get(key insolar.ID) interface{} {
	v, ok := c.Cache[key]
	if !ok {
		return nil
	}
	return v
}

func (c *Cache) Has(key insolar.ID) bool {
	_, ok := c.Cache[key]
	return ok
}

func (c *Cache) Delete(key insolar.ID) {
	delete(c.Cache, key)
}
