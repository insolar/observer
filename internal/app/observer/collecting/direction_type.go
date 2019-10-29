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

type TxDirection int

const (
	UG TxDirection = iota
	GG
	GU
	UU
)

func (d *TxDirection) String() string {
	switch *d {
	case GU:
		return "g2u"
	case UG:
		return "u2g"
	case UU:
		return "u2u"
	case GG:
		return "g2g"
	default:
		return "unknown"
	}
}
