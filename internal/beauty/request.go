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

package beauty

import (
	"encoding/hex"
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
		Arguments:  hex.EncodeToString(req.Arguments),
		Reason:     req.Reason.String(),
	}
}

func (b *Beautifier) storeRequest(request *Request) error {
	_, err := b.db.Model(request).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
