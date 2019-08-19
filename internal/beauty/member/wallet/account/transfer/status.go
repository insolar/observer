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
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/dto"
)

type txResult struct {
	status dto.Status
	fee    string
}

func parseTransferResultPayload(rec *record.Material) txResult {
	res := (*dto.Result)(rec)
	if !res.IsSuccess() {
		return txResult{dto.CANCELED, ""}
	}

	rets := res.ParsePayload().Returns
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
	return txResult{status: dto.SUCCESS, fee: fee}
}
