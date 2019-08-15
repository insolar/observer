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
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/log"
	"github.com/pkg/errors"
)

type Pulse struct {
	tableName struct{} `sql:"pulses"`

	Pulse         insolar.PulseNumber `sql:",pk"`
	PulseDate     int64               `sql:",notnull"`
	Entropy       string              `sql:",notnull"`
	RequestsCount uint32
}

func (p *Pulse) Dump(tx *pg.Tx) error {
	res, err := tx.Model(p).
		OnConflict("DO NOTHING").
		Insert()
	if err != nil {
		return errors.Wrapf(err, "failed to insert pulse %v", p)
	}
	if res.RowsAffected() < 1 {
		log.Warn(errors.Errorf("pulse inserting conflict %v", p))
	}
	return nil
}
