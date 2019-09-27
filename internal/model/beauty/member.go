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
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Member struct {
	tableName struct{} `sql:"members"`

	MemberRef        []byte `sql:",pk"`
	Balance          string `sql:",notnull"`
	MigrationAddress string
	WalletRef        []byte
	AccountState     []byte `sql:",notnull"`
	Status           string
}

func (m *Member) Dump(tx orm.DB, errorCounter prometheus.Counter) error {
	res, err := tx.Model(m).OnConflict("DO NOTHING").Insert(m)
	if err != nil {
		return errors.Wrapf(err, "failed to insert member")
	}

	if res.RowsAffected() == 0 {
		errorCounter.Inc()
		logrus.Errorf("Failed to insert member: %v", m)
	}
	return nil
}
