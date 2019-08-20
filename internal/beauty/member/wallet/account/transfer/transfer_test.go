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

package transfer

import (
	"testing"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	memberProxy "github.com/insolar/insolar/logicrunner/builtin/proxy/member"
	"github.com/stretchr/testify/require"
)

func makeCallRequest() *record.Material {
	return &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Call",
					Prototype: memberProxy.PrototypeReference,
				},
			},
		},
	}
}

func makeTransfer() []*record.Material {
	return []*record.Material{makeCallRequest()}
}

func makeTransactionMock() *pg.Tx {
	return &pg.Tx{}
}

func TestNewComposer(t *testing.T) {
	require.NotNil(t, NewComposer())
}

func TestComposer_ProcessDump(t *testing.T) {
	t.Run("whole", func(t *testing.T) {
		composer := NewComposer()

		transfer := makeTransfer()
		for _, rec := range transfer {
			composer.Process(rec)
		}

		composer.Dump(nil, nil)
	})
}
