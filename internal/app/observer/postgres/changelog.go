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

package postgres

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/observability"
)

type ChangeLogSchema struct {
	tableName struct{} `sql:"databasechangelog"`

	Id       string `sql:",pk"`
	ExecType string `sql:"exectype,notnull"`
}

type ChangeLogStorage struct {
	cfg *configuration.Configuration
	log *logrus.Logger
	db  orm.DB
}

func NewChangeLogStorage(cfg *configuration.Configuration, obs *observability.Observability, db orm.DB) *ChangeLogStorage {
	return &ChangeLogStorage{
		cfg: cfg,
		log: obs.Log(),
		db:  db,
	}
}

func (s *ChangeLogStorage) CheckVersion(currentSet string) bool {
	var err error
	set := &ChangeLogSchema{}
	err = s.db.Model(set).
		Where("exectype = ?", "EXECUTED").
		Order("orderexecuted DESC").
		Limit(1).
		Select()

	if err != nil && err != pg.ErrNoRows {
		s.log.Error(errors.Wrapf(err, "failed request to db"))
		return false
	}

	if err == pg.ErrNoRows {
		return false
	}

	if set.Id != currentSet {
		return false
	}

	return true
}
