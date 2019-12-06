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
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type MemberSchema struct {
	tableName struct{} `sql:"members"` //nolint: unused,structcheck

	MemberRef        []byte `sql:",pk"`
	Balance          string `sql:",notnull"`
	MigrationAddress string
	WalletRef        []byte
	AccountState     []byte `sql:",notnull"`
	Status           string
	AccountRef       []byte
	PublicKey        string `sql:",notnull"`
}

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
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert member %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("member_row", row).Errorf("failed to insert member")
		return errors.New("failed to insert, affected is 0")
	}
	return nil
}

func (s *MemberStorage) Update(model *observer.Balance) error {
	if model == nil {
		s.log.Warnf("trying to apply nil update balance model")
		return nil
	}

	res, err := s.db.Model(&MemberSchema{}).
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

func memberSchema(model *observer.Member) *MemberSchema {
	return &MemberSchema{
		MemberRef:        model.MemberRef.Bytes(),
		Balance:          model.Balance,
		MigrationAddress: model.MigrationAddress,
		AccountState:     model.AccountState.Bytes(),
		Status:           model.Status,
		WalletRef:        model.WalletRef.Bytes(),
		AccountRef:       model.AccountRef.Bytes(),
		PublicKey:        model.PublicKey,
	}
}
