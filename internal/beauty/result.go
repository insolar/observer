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

type Result struct {
	tableName struct{} `sql:"results"`

	ResultID string `sql:",pk,column_name:result_id"`
	Request  string
	Payload  string

	requestID insolar.ID
}

func (b *Beautifier) parseResult(id insolar.ID, res *record.Result) {
	b.rawResults[id] = &Result{
		ResultID: id.String(),
		Request:  res.Request.String(),
		//Payload:  string(res.Payload),
	}
}

func (b *Beautifier) storeResult(result *Result) error {
	_, err := b.db.Model(result).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
