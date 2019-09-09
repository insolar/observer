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

package observer

import (
	"encoding/json"
	"errors"
	"reflect"
	"runtime/debug"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
)

type Result record.Material

func CastToResult(r interface{}) *Result {
	rec, ok := r.(*Record)
	if !ok {
		log.Warnf("trying to cast %s as *observer.Record", reflect.TypeOf(r))
		return nil
	}
	return (*Result)(rec)
}

func (r *Result) IsResult() bool {
	if r == nil {
		return false
	}

	_, ok := r.Virtual.Union.(*record.Virtual_Result)
	return ok
}

func (r *Result) Request() insolar.ID {
	if !r.IsResult() {
		return insolar.ID{}
	}
	result := r.Virtual.GetResult()
	id := result.Request.GetLocal()
	if id == nil {
		return insolar.ID{}
	}
	return *id
}

func (r *Result) ParsePayload() foundation.Result {
	if r == nil {
		log.Errorf("trying to use nil dto.Result receiver")
		debug.PrintStack()
		return foundation.Result{}
	}
	payload := r.Virtual.GetResult().Payload
	if payload == nil {
		log.Warn("trying to parse nil Result.Payload")
		return foundation.Result{}
	}
	result := foundation.Result{}
	err := insolar.Deserialize(payload, &result)
	if err != nil {
		log.Warnf("failed to parse payload as foundation.Result{}")
		return foundation.Result{}
	}
	return result
}

func (r *Result) ParseFirstPayloadValue(v interface{}) {
	if !r.IsSuccess() {
		return
	}

	returns := r.ParsePayload().Returns
	data, err := json.Marshal(returns[0])
	if err != nil {
		log.Warn("failed to marshal Payload.Returns[0]")
		debug.PrintStack()
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		log.Warn("failed to unmarshal Payload.Returns[0]")
		debug.PrintStack()
	}
}

type Status string

const (
	PENDING  Status = "PENDING"
	SUCCESS  Status = "SUCCESS"
	CANCELED Status = "CANCELED"
)

func (r *Result) IsSuccess() bool {
	if !r.IsResult() {
		return false
	}

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

type CoupledResult struct {
	Request *Request
	Result  *Result
}

type ResultCollector interface {
	Collect(*Record) *CoupledResult
}
