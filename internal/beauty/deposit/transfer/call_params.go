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
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	log "github.com/sirupsen/logrus"
)

type transferCallParams struct {
	amount    string
	ethTxHash string
}

func parseTransferCallParams(request member.Request) transferCallParams {
	var (
		amount    = ""
		ethTxHash = ""
	)
	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		log.Warnf("failed to cast CallParams to map[string]interface{}")
		return transferCallParams{}
	}
	if a, ok := callParams["amount"]; ok {
		if amount, ok = a.(string); !ok {
			log.Warnf(`failed to cast CallParams["amount"] to string`)
		}
	} else {
		log.Warnf(`failed to get CallParams["amount"]`)
	}
	if t, ok := callParams["ethTxHash"]; ok {
		if ethTxHash, ok = t.(string); !ok {
			log.Warnf(`failed to cast CallParams["toMemberReference"] to string`)
		}
	} else {
		log.Warnf(`failed to get CallParams["toMemberReference"]`)
	}
	return transferCallParams{
		amount:    amount,
		ethTxHash: ethTxHash,
	}
}
