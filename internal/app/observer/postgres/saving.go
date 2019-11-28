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
	"github.com/go-pg/pg/orm"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type SavingSchema struct {
	tableName       struct{} `sql:"product_savings"`
	Ref             []byte   `sql:",pk"`
	NextPayment     int64    `sql:",notnull"`
	StartPeriodDate int64    `sql:",notnull"`
	State           []byte   `sql:",notnull"`
}

func NewSavingStorage(obs *observability.Observability, db orm.DB) *SavingStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_saving_storage_error_counter",
		Help: "",
	})
	return &SavingStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

type SavingStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func (s *SavingStorage) Insert(model *observer.NormalSaving) error {
	if model == nil {
		s.log.Warnf("trying to insert nil saving model")
		return nil
	}
	row := savingSchema(model)
	_, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert saving %v, %v", row, err.Error())
	}

	var group = GroupSchema{}

	err = s.db.Model(&group).Where("product_ref = ?", model.Reference.Bytes()).Select()
	if err != nil {
		return errors.Wrapf(err, "failed to select group %v", err.Error())
	}

	for uerRef, contributeDate := range model.NSContribute {
		_, err := s.db.Model(&UserGroupSchema{}).
			Where("group_ref=?", group.Ref).
			Where("user_ref=?", uerRef.Bytes()).
			Set("contribution_date=?", contributeDate).
			Update()
		if err != nil {
			return errors.Wrapf(err, "failed to update user_group =%v", model)
		}
	}

	return nil
}

func (s *SavingStorage) Update(model *observer.SavingUpdate) error {
	var group = GroupSchema{}
	err := s.db.Model(&group).Where("product_ref = ?", model.Reference.Bytes()).Select()
	if err != nil {
		return errors.Wrapf(err, "failed to select group %v", err.Error())
	}
	for uerRef, contributeDate := range model.NSContribute {
		_, err := s.db.Model(&UserGroupSchema{}).
			Where("group_ref=?", group.Ref).
			Where("user_ref=?", uerRef.Bytes()).
			Set("contribution_date=?", contributeDate).
			Update()
		if err != nil {
			return errors.Wrapf(err, "failed to update user_group =%v", model)
		}
	}
	_, err = s.db.Model(&SavingSchema{}).
		Where("state=?", model.PrevState.Bytes()).
		Set("next_payment=?", model.NextPaymentDate).
		Set("start_period_date=?", model.StartRoundDate).
		Set("state=?", model.SavingState.Bytes()).
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update saving model =%v", model)
	}
	return nil
}

func savingSchema(model *observer.NormalSaving) *SavingSchema {
	return &SavingSchema{
		Ref:             model.Reference.Bytes(),
		NextPayment:     model.NextPaymentDate,
		StartPeriodDate: model.StartRoundDate,
		State:           model.State.Bytes(),
	}
}
