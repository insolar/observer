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

package component

import (
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/stretchr/testify/require"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func Test_SortByType(t *testing.T) {
	var batch []*observer.Record
	batch = append(batch,
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Deactivate{},},},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Result{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_OutgoingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Activate{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Code{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Amend{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_PendingFilament{}}},
	)

	var expected []*observer.Record
	expected = append(expected,
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Code{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_PendingFilament{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_OutgoingRequest{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Activate{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Amend{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Deactivate{}}},
		&observer.Record{Virtual: record.Virtual{Union: &record.Virtual_Result{}}},
	)

	// not random but shuffled
	sort.Slice(batch, func(i, j int) bool {
		return TypeOrder(batch[i]) < TypeOrder(batch[j])
	})
	require.Equal(t, expected, batch)

	// real random
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(batch), func(i, j int) { batch[i], batch[j] = batch[j], batch[i] })
	sort.Slice(batch, func(i, j int) bool {
		return TypeOrder(batch[i]) < TypeOrder(batch[j])
	})
	require.Equal(t, expected, batch)
}
