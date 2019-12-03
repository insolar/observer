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

package observer

import (
	"context"

	"github.com/insolar/insolar/insolar"
)

type Pulse struct {
	Number    insolar.PulseNumber
	Entropy   insolar.Entropy
	Timestamp int64
	Nodes     []insolar.Node
}

type PulseStorage interface {
	Insert(*Pulse) error
	Last() (*Pulse, error)
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer.PulseFetcher -o ./ -s _mock.go -g
type PulseFetcher interface {
	Fetch(context.Context, insolar.PulseNumber) (*Pulse, error)
	FetchCurrent(ctx context.Context) (insolar.PulseNumber, error)
}
