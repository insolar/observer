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

package account

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/account"
	proxyAccount "github.com/insolar/insolar/logicrunner/builtin/proxy/account"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func IsAccountActivate(act *record.Activate) bool {
	return act.Image.Equal(*proxyAccount.PrototypeReference)
}

func IsNewAccount(rec *record.Material) bool {
	_, ok := rec.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		return false
	}
	in := rec.Virtual.GetIncomingRequest()
	if in.Method != "New" {
		return false
	}
	if in.Prototype == nil {
		return false
	}
	return in.Prototype.Equal(*proxyAccount.PrototypeReference)
}

func IsAccountAmend(amd *record.Amend) bool {
	return amd.Image.Equal(*proxyAccount.PrototypeReference)
}

func AccountBalance(rec *record.Material) string {
	memory := []byte{}
	balance := ""
	switch v := rec.Virtual.Union.(type) {
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
