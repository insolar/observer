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
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	PENDING  = "PENDING"
	SUCCESS  = "SUCCESS"
	CANCELED = "CANCELED"
)

type txResult struct {
	status string
	fee    string
}

func parseTransferResultPayload(payload []byte) txResult {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return txResult{status: "NOT_ENOUGH_PAYLOAD_PARAMS", fee: ""}
	}
	if rets[1] != nil {
		if retError, ok := rets[1].(map[string]interface{}); ok {
			if val, ok := retError["S"]; ok {
				if msg, ok := val.(string); ok {
					log.Debug(errors.New(msg))
				}
			}
			return txResult{CANCELED, ""}
		}
		log.Error(errors.New("invalid error value in member.Transfer payload"))
		return txResult{"INVALID_ERROR_VALUE", ""}
	}
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return txResult{status: "FIRST_PARAM_NOT_MAP", fee: ""}
	}
	feeInterface, ok := params["fee"]
	if !ok {
		return txResult{status: "FEE_PARAM_NOT_EXIST", fee: ""}
	}
	fee, ok := feeInterface.(string)
	if !ok {
		return txResult{status: "FEE_NOT_STRING", fee: ""}
	}
	return txResult{status: SUCCESS, fee: fee}
}
