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

package configuration

import (
	"time"

	"github.com/insolar/observer/internal/pkg/cycle"
)

type Replicator struct {
	Addr            string
	MaxTransportMsg int
	Attempts        cycle.Limit
	// Interval between fetching heavy
	AttemptInterval time.Duration
	// Using when catching up heavy on empty pulses
	FastForwardInterval time.Duration
	BatchSize           uint32
	CacheSize           int
}
