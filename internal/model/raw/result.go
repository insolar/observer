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
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
)

type Result struct {
	tableName struct{} `sql:"results"`

	ResultID string `sql:",pk,column_name:result_id"`
	Request  string
	Payload  string
}

func (r *Result) Dump(tx *pg.Tx) error {
	if err := tx.Insert(r); err != nil {
		return errors.Wrapf(err, "failed to insert result")
	}
	return nil
}