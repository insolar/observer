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

package member

import (
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/dto"
)

type memberResultParams struct {
	status           dto.Status
	migrationAddress string
	reference        string
}

func memberStatus(rec *record.Material) memberResultParams {
	res := (*dto.Result)(rec)
	if !res.IsSuccess() {
		return memberResultParams{dto.CANCELED, "", ""}
	}

	rets := res.ParsePayload().Returns
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return memberResultParams{"FIRST_PARAM_NOT_MAP", "", ""}
	}
	referenceInterface, ok := params["reference"]
	if !ok {
		return memberResultParams{dto.SUCCESS, "", ""}
	}
	reference, ok := referenceInterface.(string)
	if !ok {
		return memberResultParams{"MIGRATION_ADDRESS_NOT_STRING", "", ""}
	}

	migrationAddressInterface, ok := params["migrationAddress"]
	if !ok {
		return memberResultParams{dto.SUCCESS, "", reference}
	}
	migrationAddress, ok := migrationAddressInterface.(string)
	if !ok {
		return memberResultParams{"MIGRATION_ADDRESS_NOT_STRING", "", reference}
	}
	return memberResultParams{dto.SUCCESS, migrationAddress, reference}
}
