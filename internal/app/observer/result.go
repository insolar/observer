package observer

import (
	"encoding/json"
	"reflect"
	"runtime/debug"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/pkg/errors"
)

type Result record.Material

func CastToResult(r interface{}) (*Result, error) {
	if res, ok := r.(*Result); ok {
		return res, nil
	}
	if rec, ok := r.(*Record); ok {
		return (*Result)(rec), nil
	}
	return nil, errors.New("trying to cast %s as *observer.Record " + reflect.TypeOf(r).String())
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

func (r *Result) ParsePayload(log insolar.Logger) (foundation.Result, error) {
	if r == nil {
		// todo: throw out logger from here
		log.Errorf("trying to use nil dto.Result receiver")
		debug.PrintStack()
		return foundation.Result{}, nil
	}
	payload := r.Virtual.GetResult().Payload
	if payload == nil {
		return foundation.Result{}, nil
	}

	return ExtractFoundationResult(payload)
}

func ExtractFoundationResult(payload []byte) (foundation.Result, error) {

	// todo fix this checkup
	if payload == nil {
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

func (r *Result) ParseFirstPayloadValue(v interface{}, log insolar.Logger) {
	if !r.IsSuccess(log) {
		return
	}

	result, err := r.ParsePayload(log)
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
		log.WithField("json", string(data)).Warn("failed to unmarshal Payload.Returns[0]")
		debug.PrintStack()
	}
}

type Status string

const (
	PENDING  Status = "PENDING"
	SUCCESS  Status = "SUCCESS"
	CANCELED Status = "CANCELED"
)

func (r *Result) IsSuccess(log insolar.Logger) bool {
	if !r.IsResult() {
		return false
	}

	result, err := r.ParsePayload(log)
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
