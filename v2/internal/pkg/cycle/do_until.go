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

package cycle

import (
	"math"
	"time"
)

type Limit int

const (
	INFINITY Limit = math.MaxInt32
)

func UntilError(f func() error, interval time.Duration, attempts Limit) {
	counter := Limit(1)
	for {
		err := f()
		if err != nil {
			if counter >= attempts {
				return
			}
			time.Sleep(interval)
			continue
		}
		return
	}
}
