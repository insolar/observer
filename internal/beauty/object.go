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
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

type Object struct {
	tableName struct{} `sql:"objects"`

	ObjectID  string `sql:",pk,column_name:object_id"`
	Domain    string
	Request   string
	Memory    string
	Image     string
	Parent    string
	PrevState string

	requestID insolar.ID
}

func (b *Beautifier) parseActivate(id insolar.ID, act *record.Activate) {
	b.rawObjects[id] = &Object{
		ObjectID: id.String(),
		Domain:   act.Domain.String(),
		Request:  act.Request.String(),
		//Memory:   string(act.Memory),
		Image:  act.Image.String(),
		Parent: act.Parent.String(),
	}
}

func (b *Beautifier) parseAmend(id insolar.ID, amend *record.Amend) {
	b.rawObjects[id] = &Object{
		ObjectID: id.String(),
		Domain:   amend.Domain.String(),
		Request:  amend.Request.String(),
		//Memory:   string(amend.Memory),
		Image:  amend.Image.String(),
		Parent: amend.PrevState.String(),
	}
}

func (b *Beautifier) parseDeactivate(id insolar.ID, deact *record.Deactivate) {
	b.rawObjects[id] = &Object{
		ObjectID: id.String(),
		Domain:   deact.Domain.String(),
		Request:  deact.Request.String(),
		Parent:   deact.PrevState.String(),
	}
}

func (b *Beautifier) storeObject(object *Object) error {
	_, err := b.db.Model(object).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
