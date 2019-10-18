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

	"github.com/insolar/insolar/application/builtin/contract/member"
	depositProxy "github.com/insolar/insolar/application/builtin/proxy/deposit"
	memberProxy "github.com/insolar/insolar/application/builtin/proxy/member"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
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
	t.Run("not_request", func(t *testing.T) {
		request := (*Request)(makeResultWith([]byte{1, 2, 3}))

		require.False(t, request.IsMemberCall())
	})

	t.Run("nil_prototype", func(t *testing.T) {
		t.Skip()
		request := makeRequestWith("Call", nil, nil)

		require.False(t, request.IsMemberCall())
	})

	t.Run("not_member", func(t *testing.T) {
		t.Skip()
		request := makeRequestWith("Call", depositProxy.PrototypeReference, nil)

		require.False(t, request.IsMemberCall())
	})

	t.Run("not_call", func(t *testing.T) {
		request := makeRequestWith("test", memberPrototype, nil)

		require.False(t, request.IsMemberCall())
	})

	t.Run("call", func(t *testing.T) {
		request := makeRequestWith("Call", memberPrototype, nil)

		require.True(t, request.IsMemberCall())
	})
}

func TestRequest_ParseMemberCallArguments(t *testing.T) {
	memberPrototype := memberProxy.PrototypeReference
	emptyResult := member.Request{}

	t.Run("empty_args", func(t *testing.T) {
		request := makeRequestWith("Call", memberPrototype, nil)

		actual := request.ParseMemberCallArguments()
		require.Equal(t, emptyResult, actual)
	})
}

func TestRequest_ParseMemberContractCallParams(t *testing.T) {

}
