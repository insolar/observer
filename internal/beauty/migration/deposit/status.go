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

package deposit

import (
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/dto"
)

type depositStatus struct {
	status    dto.Status
	memberRef string
}

func parseMemberRef(rec *record.Material) depositStatus {
	res := (*dto.Result)(rec)
	if !res.IsSuccess() {
		return depositStatus{dto.CANCELED, ""}
	}

	rets := res.ParsePayload().Returns

	resultMap, ok := rets[0].(map[string]interface{})
	if !ok {
		return depositStatus{"INVALID_RESULT_VALUE", ""}
	}

	memberRefInterface, ok := resultMap["memberReference"]
	if !ok {
		return depositStatus{"HAS_NOT_MEMBER_REFERENCE", ""}
	}
	memberRef, ok := memberRefInterface.(string)
	if !ok {
		return depositStatus{"MEMBER_REFERENCE_IS_NOT_STRING", ""}
	}

	return depositStatus{dto.SUCCESS, memberRef}
}
