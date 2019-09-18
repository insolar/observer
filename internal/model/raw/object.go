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
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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
	Type      string
}

func (o *Object) Dump(tx orm.DB, errorCounter prometheus.Counter) error {
	res, err := tx.Model(o).OnConflict("DO NOTHING").Insert(o)
	if err != nil {
		return errors.Wrapf(err, "failed to insert object")
	}

	if res.RowsAffected() == 0 {
		errorCounter.Inc()
		logrus.Errorf("Failed to insert object: %v", o)
	}
	return nil
}
