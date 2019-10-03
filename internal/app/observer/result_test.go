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
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"
)

func makeResultWith(payload []byte) *Result {
	return &Result{Virtual: record.Virtual{Union: &record.Virtual_Result{Result: &record.Result{Payload: payload}}}}
}

func TestResult_ParsePayload(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var result *Result
		res, err := result.ParsePayload()
		require.NoError(t, err)
		require.Equal(t, foundation.Result{}, res)
	})

	t.Run("empty", func(t *testing.T) {
		res := makeResultWith(nil)
		result, err := res.ParsePayload()
		require.NoError(t, err)
		require.Equal(t, foundation.Result{}, result)
	})

	t.Run("nonsense", func(t *testing.T) {
		res := makeResultWith([]byte{1, 2, 3})
		result, err := res.ParsePayload()
		require.Error(t, err)
		require.Equal(t, foundation.Result{}, result)
	})

	t.Run("ordinary", func(t *testing.T) {
		initial := foundation.Result{
			Error:   &foundation.Error{S: "request error msg"},
			Returns: []interface{}{"return value", &foundation.Error{S: "contract error msg"}},
		}
		expected := foundation.Result{
			Error:   &foundation.Error{S: "request error msg"},
			Returns: []interface{}{"return value", &foundation.Error{S: "contract error msg"}},
		}
		payload, err := insolar.Serialize(initial)
		require.NoError(t, err)
		res := makeResultWith(payload)
		result, err := res.ParsePayload()
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}

func TestResult_IsSuccess(t *testing.T) {
	t.Run("outside_error", func(t *testing.T) {
		outsideContractError, err := insolar.Serialize(&foundation.Result{
			Error:   &foundation.Error{S: "request error msg"},
			Returns: []interface{}{"return value", nil},
		})
		require.NoError(t, err)
		outsideErrorResult := makeResultWith(outsideContractError)

		require.False(t, outsideErrorResult.IsSuccess())
	})

	t.Run("inside_error", func(t *testing.T) {
		insideContractError, err := insolar.Serialize(&foundation.Result{
			Error:   nil,
			Returns: []interface{}{"return value", &foundation.Error{S: "contract error msg"}},
		})
		require.NoError(t, err)
		insideErrorResult := makeResultWith(insideContractError)

		require.False(t, insideErrorResult.IsSuccess())
	})

	t.Run("success", func(t *testing.T) {
		success, err := insolar.Serialize(&foundation.Result{
			Error:   nil,
			Returns: []interface{}{"return value", nil},
		})
		require.NoError(t, err)
		successResult := makeResultWith(success)

		require.True(t, successResult.IsSuccess())
	})
}
