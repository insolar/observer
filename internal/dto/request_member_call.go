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
	"reflect"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member/signer"
	proxyMember "github.com/insolar/insolar/logicrunner/builtin/proxy/member"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (r *Request) IsMemberCall() bool {
	if r == nil {
		return false
	}

	v, ok := r.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		log.Errorf("trying to use %s as IncomingRequest", reflect.TypeOf(r.Virtual.Union).String())
		return false
	}

	req := v.IncomingRequest
	if req.Prototype == nil {
		return false
	}

	if !req.Prototype.Equal(*proxyMember.PrototypeReference) {
		return false
	}
	return req.Method == "Call"
}

func (r *Request) ParseMemberCallArguments() member.Request {
	if !r.IsMemberCall() {
		log.Errorf("trying to parse member call arguments of not member call")
		return member.Request{}
	}

	in := r.Virtual.GetIncomingRequest().Arguments
	if in == nil {
		log.Warnf("member call arguments is nil")
		return member.Request{}
	}
	var args []interface{}
	err := insolar.Deserialize(in, &args)
	if err != nil {
		log.Warn(errors.Wrapf(err, "failed to deserialize request arguments"))
		return member.Request{}
	}

	request := member.Request{}
	if len(args) > 0 {
		if rawRequest, ok := args[0].([]byte); ok {
			var (
				pulseTimeStamp int64
				signature      string
				raw            []byte
			)
			err = signer.UnmarshalParams(rawRequest, &raw, &signature, &pulseTimeStamp)
			if err != nil {
				log.Warn(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				log.Warn(errors.Wrapf(err, "failed to unmarshal json member request"))
				return member.Request{}
			}
		}
	}
	return request
}
