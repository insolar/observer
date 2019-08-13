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

const (
	PENDING  = "PENDING"
	SUCCESS  = "SUCCESS"
	CANCELED = "CANCELED"
)

type memberResultParams struct {
	status           string
	migrationAddress string
	reference        string
}

func memberStatus(payload []byte) memberResultParams {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return memberResultParams{"NOT_ENOUGH_PAYLOAD_PARAMS", "", ""}
	}
	if retError, ok := rets[1].(error); ok {
		if retError != nil {
			return memberResultParams{CANCELED, "", ""}
		}
	}
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return memberResultParams{"FIRST_PARAM_NOT_MAP", "", ""}
	}
	referenceInterface, ok := params["reference"]
	if !ok {
		return memberResultParams{SUCCESS, "", ""}
	}
	reference, ok := referenceInterface.(string)
	if !ok {
		return memberResultParams{"MIGRATION_ADDRESS_NOT_STRING", "", ""}
	}

	migrationAddressInterface, ok := params["migrationAddress"]
	if !ok {
		return memberResultParams{SUCCESS, "", reference}
	}
	migrationAddress, ok := migrationAddressInterface.(string)
	if !ok {
		return memberResultParams{"MIGRATION_ADDRESS_NOT_STRING", "", reference}
	}
	return memberResultParams{SUCCESS, migrationAddress, reference}
}
