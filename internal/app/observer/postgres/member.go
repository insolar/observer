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

type MemberStorage struct {
	log          insolar.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewMemberStorage(obs *observability.Observability, db orm.DB) *MemberStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_member_storage_error_counter",
		Help: "",
	})
	return &MemberStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *MemberStorage) Insert(model *observer.Member) error {
	if model == nil {
		s.log.Warnf("trying to insert nil member model")
		return nil
	}
	row := memberSchema(model)
	log := s.log.WithField("memberRef", model.MemberRef.String())
	res, err := s.db.Model(row).Insert()
	if err != nil {
		return errors.Wrapf(err, "failed to insert member %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("member_row", row).Errorf("failed to insert member")
		return errors.New("failed to insert, affected is 0")
	}

	log.Debug("inserted member")

	return nil
}

func (s *MemberStorage) Update(model *observer.Balance) error {
	if model == nil {
		s.log.Warnf("trying to apply nil update balance model")
		return nil
	}

	res, err := s.db.Model(&observer.Member{}).
		Where("account_state=?", model.PrevState.Bytes()).
		Set("balance=?,account_state=?", model.Balance, model.AccountState.Bytes()).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update member balance upd=%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).Errorf("failed to update member")
		// TODO: uncomment it
		// return errors.New("failed to update, affected is 0")
	}
	return nil
}

func memberSchema(model *observer.Member) *models.Member {
	return &models.Member{
		Reference:        model.MemberRef.Bytes(),
		Balance:          model.Balance,
		MigrationAddress: model.MigrationAddress,
		AccountState:     model.AccountState.Bytes(),
		Status:           model.Status,
		WalletReference:  model.WalletRef.Bytes(),
		AccountReference: model.AccountRef.Bytes(),
		PublicKey:        model.PublicKey,
	}
}
