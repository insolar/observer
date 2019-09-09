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

import (
	"encoding/json"
	"testing"

	"github.com/insolar/insolar/api/requester"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/account"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	proxyAccount "github.com/insolar/insolar/logicrunner/builtin/proxy/account"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/internal/app/observer"
)

func makeAccountActivate(pulse insolar.PulseNumber, balance string, requestRef insolar.Reference) *observer.Record {
	acc := &account.Account{
		Balance: balance,
	}
	memory, err := insolar.Serialize(acc)
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: requestRef,
					Memory:  memory,
					Image:   *proxyAccount.PrototypeReference,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeNewAccountRequest(pulse insolar.PulseNumber, reason insolar.Reference) *observer.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "New",
					Arguments: args,
					Prototype: proxyAccount.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeMemberCreateCall(pulse insolar.PulseNumber) *observer.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite:   "member.create",
			CallParams: nil,
		},
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		panic("failed to marshal request")
	}
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{requestBody, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &record.Material{
		ID: gen.IDWithPulse(pulse),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Call",
					Arguments: args,
				},
			},
		},
	}
	return (*observer.Record)(rec)
}

func makeMember() ([]*observer.Member, []*observer.Record) {
	pn := insolar.GenesisPulse.PulseNumber
	balance := "42"
	memberRef := *insolar.NewReference(gen.IDWithPulse(pn))
	out := makeOutgouingRequest()
	call := makeMemberCreateCall(pn)
	callRef := *insolar.NewReference(call.ID)
	newAccount := makeNewAccountRequest(pn, callRef)
	newAccountRef := *insolar.NewReference(newAccount.ID)
	accountActivate := makeAccountActivate(pn, balance, newAccountRef)
	records := []*observer.Record{
		out,
		makeResultWith(out.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		call,
		makeResultWith(call.ID, &foundation.Result{Returns: []interface{}{&member.CreateResponse{
			Reference: memberRef.String(),
		}, nil}}),
		newAccount,
		makeResultWith(newAccount.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
		accountActivate,
	}
	member := &observer.Member{
		MemberRef:        memberRef,
		Balance:          balance,
		MigrationAddress: "",
		AccountState:     accountActivate.ID,
	}
	return []*observer.Member{member}, records
}

func TestMemberCollector_Collect(t *testing.T) {
	collector := NewMemberCollector()

	expected, records := makeMember()
	var actual []*observer.Member
	for _, r := range records {
		member := collector.Collect(r)
		if member != nil {
			actual = append(actual, member)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
