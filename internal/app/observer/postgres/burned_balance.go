// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

type BurnedBalanceStorage struct {
	log          insolar.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewBurnedBalanceStorage(obs *observability.Observability, db orm.DB) *BurnedBalanceStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_burned_balance_storage_error_counter",
		Help: "",
	})
	return &BurnedBalanceStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *BurnedBalanceStorage) Insert(model *observer.BurnedBalance) error {
	if model == nil {
		s.log.Warnf("trying to insert nil burned balance model")
		return nil
	}
	row := burnedBalanceSchema(model)
	res, err := s.db.Model(row).Insert()
	if err != nil {
		return errors.Wrapf(err, "failed to insert burned balance %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("burned_balance_row", row).Errorf("failed to insert burned balance")
		return errors.New("failed to insert, affected is 0")
	}

	s.log.Debug("inserted burned balance")

	return nil
}

func (s *BurnedBalanceStorage) Update(model *observer.BurnedBalance) error {
	if model == nil {
		s.log.Warnf("trying to apply nil update burned balance model")
		return nil
	}

	res, err := s.db.Model(&models.BurnedBalance{}).
		Where("account_state=?", model.PrevState.Bytes()).
		Set("balance=?,account_state=?", model.Balance, model.AccountState.Bytes()).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update burned balance upd=%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).Errorf("failed to update burned balance")
	}
	return nil
}

func burnedBalanceSchema(model *observer.BurnedBalance) *models.BurnedBalance {
	return &models.BurnedBalance{
		Balance:      model.Balance,
		AccountState: model.AccountState.Bytes(),
	}
}
