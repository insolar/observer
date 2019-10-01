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
	"encoding/json"
	"runtime/debug"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	log "github.com/sirupsen/logrus"
)

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