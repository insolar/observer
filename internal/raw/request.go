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

package raw

import (
	"encoding/hex"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/model/raw"
)

func parseRequest(rec *record.Material) *raw.Request {
	id := rec.ID
	req := rec.Virtual.GetIncomingRequest()
	var base, object, prototype = "", "", ""
	if nil != req.Base {
		base = req.Base.String()
	}
	if nil != req.Object {
		object = req.Object.String()
	}
	if nil != req.Prototype {
		prototype = req.Prototype.String()
	}
	return &raw.Request{
		RequestID:  insolar.NewReference(id).String(),
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
