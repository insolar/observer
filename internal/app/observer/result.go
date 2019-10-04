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
	"reflect"
	"runtime/debug"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Result record.Material

func CastToResult(r interface{}) *Result {
	if res, ok := r.(*Result); ok {
		return res
	}
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

func (r *Result) ParsePayload() (foundation.Result, error) {
	if r == nil {
		log.Errorf("trying to use nil dto.Result receiver")
		debug.PrintStack()
		return foundation.Result{}, nil
	}
	payload := r.Virtual.GetResult().Payload
	if payload == nil {
		log.Warn("trying to parse nil Result.Payload")
		return foundation.Result{}, nil
	}

	var firstValue interface{}
	var contractErr *foundation.Error
	requestErr, err := foundation.UnmarshalMethodResult(payload, &firstValue, &contractErr)

	if err != nil {
		return foundation.Result{}, errors.Wrap(err, "failed to unmarshal result payload")
	}

	result := foundation.Result{
		Error:   requestErr,
		Returns: []interface{}{firstValue, contractErr},
	}
	return result, nil
}

func (r *Result) ParseFirstPayloadValue(v interface{}) {
	if !r.IsSuccess() {
		return
	}

	result, err := r.ParsePayload()
	if err != nil {
		return
	}
	returns := result.Returns
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

	result, err := r.ParsePayload()
	if err != nil {
		return false
	}
	if result.Error != nil {
		return false
	}

	for i := 0; i < len(result.Returns); i++ {
		if e, ok := result.Returns[i].(*foundation.Error); ok {
			if e != nil {
				return false
			}
		}
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
