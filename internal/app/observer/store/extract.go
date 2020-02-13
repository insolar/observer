// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package store

import (
	"errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

func ExtractRequestData(rec *record.Material) (insolar.ID, insolar.ID, error) {
	virtual := record.Unwrap(&rec.Virtual)
	if req, isRequest := virtual.(record.Request); isRequest {
		return rec.ID, *req.ReasonRef().GetLocal(), nil
	}
	return insolar.ID{}, insolar.ID{}, errors.New("not a request")
}

func RequestID(rec *record.Material) (insolar.ID, error) {
	virtual := record.Unwrap(&rec.Virtual)
	var (
		id  insolar.ID
		err error
	)
	switch r := virtual.(type) {
	case record.Request:
		id = rec.ID
	case *record.Result:
		id = *r.Request.GetLocal()
	case *record.Activate:
		id = *r.Request.GetLocal()
	case *record.Amend:
		id = *r.Request.GetLocal()
	case *record.Deactivate:
		id = *r.Request.GetLocal()
	default:
		err = errors.New("unknown record")
	}
	if err != nil {
		return id, err
	}
	if id.IsEmpty() {
		return id, errors.New("empty record ID")
	}
	return id, nil
}

func ReasonID(rec *record.Material) (insolar.ID, error) {
	virtual := record.Unwrap(&rec.Virtual)
	if request, ok := virtual.(record.Request); ok {
		return *request.ReasonRef().GetLocal(), nil
	}
	return insolar.ID{}, errors.New("no a request")
}
