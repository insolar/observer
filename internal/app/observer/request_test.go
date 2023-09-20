package observer

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/mainnet/application/builtin/contract/member"
	depositProxy "github.com/insolar/mainnet/application/builtin/proxy/deposit"
	memberProxy "github.com/insolar/mainnet/application/builtin/proxy/member"
	"github.com/stretchr/testify/require"
)

func makeRequestWith(method string, prototype *insolar.Reference, args []byte) *Request {
	return &Request{Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{IncomingRequest: &record.IncomingRequest{
		Method:    method,
		Prototype: prototype,
		Arguments: args,
	}}}}
}

func TestRequest_IsCall(t *testing.T) {
	memberPrototype := memberProxy.PrototypeReference
	log := inslogger.FromContext(inslogger.TestContext(t))
	t.Run("not_request", func(t *testing.T) {
		request := (*Request)(makeResultWith([]byte{1, 2, 3}))

		require.False(t, request.IsMemberCall(log))
	})

	t.Run("nil_prototype", func(t *testing.T) {
		t.Skip()
		request := makeRequestWith("Call", nil, nil)

		require.False(t, request.IsMemberCall(log))
	})

	t.Run("not_member", func(t *testing.T) {
		t.Skip()
		request := makeRequestWith("Call", depositProxy.PrototypeReference, nil)

		require.False(t, request.IsMemberCall(log))
	})

	t.Run("not_call", func(t *testing.T) {
		request := makeRequestWith("test", memberPrototype, nil)

		require.False(t, request.IsMemberCall(log))
	})

	t.Run("call", func(t *testing.T) {
		request := makeRequestWith("Call", memberPrototype, nil)

		require.True(t, request.IsMemberCall(log))
	})
}

func TestRequest_ParseMemberCallArguments(t *testing.T) {
	memberPrototype := memberProxy.PrototypeReference
	emptyResult := member.Request{}
	log := inslogger.FromContext(inslogger.TestContext(t))

	t.Run("empty_args", func(t *testing.T) {
		request := makeRequestWith("Call", memberPrototype, nil)

		actual := request.ParseMemberCallArguments(log)
		require.Equal(t, emptyResult, actual)
	})
}

func TestRequest_ParseMemberContractCallParams(t *testing.T) {

}
