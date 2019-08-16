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
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/dto"
)

const (
	PENDING  = "PENDING"
	SUCCESS  = "SUCCESS"
	CANCELED = "CANCELED"
)

type depositStatus struct {
	status    string
	memberRef string
}

func parseMemberRef(rec *record.Material) depositStatus {
	res := (*dto.Result)(rec)
	rets := res.ParsePayload().Returns
	if len(rets) < 2 {
		return depositStatus{"NOT_ENOUGH_PAYLOAD_PARAMS", ""}
	}

	if rets[1] != nil {
		if retError, ok := rets[1].(map[string]interface{}); ok {
			if val, ok := retError["S"]; ok {
				if msg, ok := val.(string); ok {
					log.Debug(errors.New(msg))
				}
			}
			return depositStatus{CANCELED, ""}
		}
		log.Error(errors.New("invalid error value in GetMigrationAddress payload"))
		return depositStatus{"INVALID_ERROR_VALUE", ""}
	}

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

	return depositStatus{SUCCESS, memberRef}
}
