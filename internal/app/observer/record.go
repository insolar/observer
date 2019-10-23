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
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

type Record record.Material

type RecordStorage interface {
	Last() *Record
	Count(insolar.PulseNumber) uint32
	Insert(*Record) error
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer.RecordFetcher -o ./ -s _mock.go -g
type RecordFetcher interface {
	Fetch(pulse insolar.PulseNumber) ([]*Record, insolar.PulseNumber, error)
}

func (r *Record) Marshal() ([]byte, error) {
	return (*record.Material)(r).Marshal()
}

func (r *Record) Unmarshal(data []byte) error {
	return (*record.Material)(r).Unmarshal(data)
}
