package observer

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/require"
)

func makeResultWith(payload []byte) *Result {
	return &Result{Virtual: record.Virtual{Union: &record.Virtual_Result{Result: &record.Result{Payload: payload}}}}
}

func TestResult_ParsePayload(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))
	t.Run("nil", func(t *testing.T) {
		var result *Result
		res, err := result.ParsePayload(log)
		require.NoError(t, err)
		require.Equal(t, foundation.Result{}, res)
	})

	t.Run("empty", func(t *testing.T) {
		res := makeResultWith(nil)
		result, err := res.ParsePayload(log)
		require.NoError(t, err)
		require.Equal(t, foundation.Result{}, result)
	})

	t.Run("nonsense", func(t *testing.T) {
		res := makeResultWith([]byte{1, 2, 3})
		result, err := res.ParsePayload(log)
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
		result, err := res.ParsePayload(log)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}

func TestResult_IsSuccess(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))
	t.Run("outside_error", func(t *testing.T) {
		outsideContractError, err := insolar.Serialize(&foundation.Result{
			Error:   &foundation.Error{S: "request error msg"},
			Returns: []interface{}{"return value", nil},
		})
		require.NoError(t, err)
		outsideErrorResult := makeResultWith(outsideContractError)

		require.False(t, outsideErrorResult.IsSuccess(log))
	})

	t.Run("inside_error", func(t *testing.T) {
		insideContractError, err := insolar.Serialize(&foundation.Result{
			Error:   nil,
			Returns: []interface{}{"return value", &foundation.Error{S: "contract error msg"}},
		})
		require.NoError(t, err)
		insideErrorResult := makeResultWith(insideContractError)

		require.False(t, insideErrorResult.IsSuccess(log))
	})

	t.Run("success", func(t *testing.T) {
		success, err := insolar.Serialize(&foundation.Result{
			Error:   nil,
			Returns: []interface{}{"return value", nil},
		})
		require.NoError(t, err)
		successResult := makeResultWith(success)

		require.True(t, successResult.IsSuccess(log))
	})
}
