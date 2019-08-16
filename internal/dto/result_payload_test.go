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

package dto

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"
)

func makeWith(payload []byte) *Result {
	return &Result{Virtual: record.Virtual{Union: &record.Virtual_Result{Result: &record.Result{Payload: payload}}}}
}

func TestParse(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var result *Result
		require.Equal(t, foundation.Result{}, result.ParsePayload())
	})

	t.Run("empty", func(t *testing.T) {
		res := makeWith([]byte{})
		require.Equal(t, foundation.Result{}, res.ParsePayload())
	})

	t.Run("nonsense", func(t *testing.T) {
		res := makeWith([]byte{1, 2, 3})
		require.Equal(t, foundation.Result{}, res.ParsePayload())
	})

	t.Run("ordinary", func(t *testing.T) {
		initial := foundation.Result{
			Error:   &foundation.Error{S: "request error msg"},
			Returns: []interface{}{"return value", &foundation.Error{S: "contract error msg"}},
		}
		expected := foundation.Result{
			Error:   &foundation.Error{S: "request error msg"},
			Returns: []interface{}{"return value", map[string]interface{}{"S": "contract error msg"}},
		}
		payload, err := insolar.Serialize(initial)
		require.NoError(t, err)
		res := makeWith(payload)
		require.Equal(t, expected, res.ParsePayload())
	})
}
