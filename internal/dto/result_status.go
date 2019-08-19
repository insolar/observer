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

package dto

import (
	"errors"
	"reflect"

	log "github.com/sirupsen/logrus"
)

type Status string

const (
	PENDING  Status = "PENDING"
	SUCCESS  Status = "SUCCESS"
	CANCELED Status = "CANCELED"
)

func (r *Result) IsSuccess() bool {
	result := r.ParsePayload()
	if result.Error != nil {
		return false
	}

	if len(result.Returns) < 2 {
		log.Warn("in parsed Result.Payload as foundation.Result, field Returns has less than 2 values")
		return false
	}

	// result.Returns[1] should contains serialized error of contract execution
	ret1 := result.Returns[1]
	if ret1 != nil {
		errMap, ok := ret1.(map[string]interface{})
		if !ok {
			log.Warn("error in foundation.Result.Returns[1] is not serialized as map")
			return false
		}

		strRepresentation, ok := errMap["S"]
		if !ok {
			log.Warn(`error in foundation.Result.Returns[1] is serialized as map but didn't has "S" value`)
			return false
		}

		msg, ok := strRepresentation.(string)
		if !ok {
			log.Warnf(`error in foundation.Result.Returns[1] is serialized as map, has "S" value, but value is not string type (actual type: %s)'`,
				reflect.TypeOf(strRepresentation))
			return false
		}

		log.Debug(errors.New(msg))
		return false
	}
	return true
}
