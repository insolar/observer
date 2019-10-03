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

package filtering

import (
	"github.com/insolar/observer/internal/app/observer"
)

type WastingFilter struct{}

func NewWastingFilter() *WastingFilter {
	return &WastingFilter{}
}

func (*WastingFilter) Filter(wastings map[string]*observer.Wasting, addresses map[string]*observer.MigrationAddress) {
	// We try to apply migration address wasting in memory.
	for key, wasting := range wastings {
		addr, ok := addresses[wasting.Addr]
		if !ok {
			continue
		}
		addr.Wasted = true
		delete(wastings, key)
	}
}
