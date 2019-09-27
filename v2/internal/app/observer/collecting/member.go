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
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/account"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	proxyAccount "github.com/insolar/insolar/logicrunner/builtin/proxy/account"

	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/internal/pkg/panic"
)

type MemberCollector struct {
	collector *BoundCollector
}

func NewMemberCollector() *MemberCollector {
	collector := NewBoundCollector(isMemberCreateRequest, successResult, isNewAccount, isAccountActivate)
	return &MemberCollector{
		collector: collector,
	}
}

func (c *MemberCollector) Collect(rec *observer.Record) *observer.Member {
	defer panic.Catch("member_collector")

	if rec == nil {
		return nil
	}
	couple := c.collector.Collect(rec)
	if couple == nil {
		return nil
	}

	m, err := c.build(couple.Activate, couple.Result)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build member"))
		return nil
	}
	return m
}

func isMemberCreateRequest(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}
	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	switch args.Params.CallSite {
	case "member.create", "member.migrationCreate":
		return true
	}
	return false
}

func successResult(chain interface{}) bool {
	result := observer.CastToResult(chain)
	return result.IsSuccess()
}

func isNewAccount(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}
	in := request.Virtual.GetIncomingRequest()
	if in.Method != "New" {
		return false
	}
	if in.Prototype == nil {
		return false
	}
	return in.Prototype.Equal(*proxyAccount.PrototypeReference)
}

func isAccountActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()
	return act.Image.Equal(*proxyAccount.PrototypeReference)
}

func (c *MemberCollector) build(act *observer.Activate, res *observer.Result) (*observer.Member, error) {
	if res == nil || act == nil {
		return nil, errors.New("trying to create member from noncomplete builder")
	}

	if res.Virtual.GetResult().Payload == nil {
		return nil, errors.New("member creation result payload is nil")
	}
	response := &member.MigrationCreateResponse{}
	res.ParseFirstPayloadValue(response)

	balance := accountBalance((*observer.Record)(act))
	ref, err := insolar.NewReferenceFromBase58(response.Reference)
	if err != nil || ref == nil {
		return nil, errors.New("invalid member reference")
	}
	return &observer.Member{
		MemberRef:        *ref,
		Balance:          balance,
		MigrationAddress: response.MigrationAddress,
		AccountState:     act.ID,
	}, nil
}

func accountBalance(act *observer.Record) string {
	memory := []byte{}
	balance := ""
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get account memory"))
	}

	if memory == nil {
		log.Warn(errors.New("account memory is nil"))
		return "0"
	}

	acc := account.Account{}
	if err := insolar.Deserialize(memory, &acc); err != nil {
		log.Error(errors.New("failed to deserialize account memory"))
	} else {
		balance = acc.Balance
	}
	return balance
}
