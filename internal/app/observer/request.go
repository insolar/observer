// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package observer

import (
	"encoding/json"
	"reflect"
	"runtime/debug"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/pkg/errors"
)

type Request record.Material

type RequestCollector interface {
	Collect(*Record)
}

func CastToRequest(r interface{}, logger insolar.Logger) *Request {
	if req, ok := r.(*Request); ok {
		return req
	}
	rec, ok := r.(*Record)
	if !ok {
		logger.Warnf("trying to cast %s as *observer.Record", reflect.TypeOf(r))
		return nil
	}
	return (*Request)(rec)
}

func (r *Request) Reason() insolar.ID {
	if r == nil {
		return insolar.ID{}
	}

	if !r.IsIncoming() {
		return insolar.ID{}
	}

	req := r.Virtual.GetIncomingRequest()
	id := req.Reason.GetLocal()
	if id == nil {
		return insolar.ID{}
	}
	return *id
}

func (r *Request) IsIncoming() bool {
	if r == nil {
		return false
	}
	_, ok := r.Virtual.Union.(*record.Virtual_IncomingRequest)
	return ok
}

func (r *Request) IsOutgoing() bool {
	if r == nil {
		return false
	}

	_, ok := r.Virtual.Union.(*record.Virtual_OutgoingRequest)
	return ok
}

func (r *Request) IsMemberCall(logger insolar.Logger) bool {
	if r == nil {
		logger.Errorf("trying to use nil dto.Request receiver")
		debug.PrintStack()
		return false
	}

	v, ok := r.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		logger.Errorf("trying to use %s as IncomingRequest", reflect.TypeOf(r.Virtual.Union).String())
		debug.PrintStack()
		return false
	}

	req := v.IncomingRequest
	// TODO: uncomment Prototype check
	// if req.Prototype == nil {
	// 	return false
	// }
	// if !req.Prototype.Equal(*proxyMember.PrototypeReference) {
	// 	return false
	// }
	return req.Method == "Call"
}

func (r *Request) ParseMemberCallArguments(logger insolar.Logger) member.Request {
	if !r.IsMemberCall(logger) {
		logger.Errorf("trying to parse member call arguments of not member call")
		debug.PrintStack()
		return member.Request{}
	}

	in := r.Virtual.GetIncomingRequest().Arguments
	if in == nil {
		logger.Warnf("member call arguments is nil")
		return member.Request{}
	}
	var args []interface{}
	err := insolar.Deserialize(in, &args)
	if err != nil {
		logger.Warn(errors.Wrapf(err, "failed to deserialize request arguments"))
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
			err = insolar.Deserialize(rawRequest, []interface{}{&raw, &signature, &pulseTimeStamp})
			if err != nil {
				logger.Warn(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				logger.Warn(errors.Wrapf(err, "failed to unmarshal json member request"))
				return member.Request{}
			}
		}
	}
	return request
}

func (r *Request) ParseMemberContractCallParams(v interface{}, logger insolar.Logger) {
	if !r.IsMemberCall(logger) {
		return
	}
	args := r.ParseMemberCallArguments(logger)
	data, err := json.Marshal(args.Params.CallParams)
	if err != nil {
		logger.Warn("failed to marshal CallParams")
		debug.PrintStack()
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		logger.Warn("failed to unmarshal CallParams")
		debug.PrintStack()
	}
}
