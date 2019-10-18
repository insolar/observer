package store

import (
	"errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

func RequestID(rec *record.Material) (insolar.ID, error) {
	virtual := record.Unwrap(&rec.Virtual)
	switch r := virtual.(type) {
	case record.Request:
		return rec.ID, nil
	case *record.Result:
		return *r.Request.GetLocal(), nil
	case *record.Activate:
		return *r.Request.GetLocal(), nil
	case *record.Amend:
		return *r.Request.GetLocal(), nil
	case *record.Deactivate:
		return *r.Request.GetLocal(), nil
	default:
		return insolar.ID{}, errors.New("unknown record")
	}
}

func ReasonID(rec *record.Material) (insolar.ID, error) {
	virtual := record.Unwrap(&rec.Virtual)
	if request, ok := virtual.(record.Request); ok {
		return *request.ReasonRef().GetLocal(), nil
	}
	return insolar.ID{}, errors.New("no a request")
}
