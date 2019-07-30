package beauty

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

type Request struct {
	tableName struct{} `sql:"requests"`

	RequestID  string `sql:",pk,column_name:request_id"`
	Caller     string
	ReturnMode string
	Base       string
	Object     string
	Prototype  string
	Method     string
	Arguments  string
	Reason     string

	requestID insolar.ID
}

func (b *Beautifier) parseRequest(id insolar.ID, req *record.IncomingRequest) {
	var base, object, prototype = "", "", ""
	if nil != req.Base {
		base = req.Base.String()
	}
	if nil != req.Object {
		object = req.Object.String()
	}
	if nil != req.Prototype {
		object = req.Prototype.String()
	}
	b.rawRequests[id] = &Request{
		RequestID:  id.String(),
		Caller:     req.Caller.String(),
		ReturnMode: req.ReturnMode.String(),
		Base:       base,
		Object:     object,
		Prototype:  prototype,
		Method:     req.Method,
		//Arguments:  string(req.Arguments),
		Reason: req.Reason.String(),
	}
}

func (b *Beautifier) storeRequest(request *Request) error {
	_, err := b.db.Model(request).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
