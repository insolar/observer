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

type UserSchema struct {
	tableName struct{} `sql:"users"`

	Ref    []byte `sql:",pk"`
	KYC    bool   `sql:",notnull"`
	Public string `sql:",notnull"`
	Status string `sql:",notnull"`
	State  []byte `sql:",notnull"`
}

func NewUserStorage(obs *observability.Observability, db orm.DB) *UserStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_user_storage_error_counter",
		Help: "",
	})
	return &UserStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

type UserStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func (s *UserStorage) Insert(model *observer.User) error {
	if model == nil {
		s.log.Warnf("trying to insert nil user model")
		return nil
	}
	row := userSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert user %v, %v", row, err.Error())
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("user_row", row).
			Errorf("failed to insert user")
	}
	return nil
}

func (s *UserStorage) Update(model *observer.UserKYC) error {
	if model == nil {
		s.log.Warnf("trying to apply nil update user model")
		return nil
	}

	res, err := s.db.Model(&UserSchema{}).
		Where("state=?", model.PrevState.Bytes()).
		Set("kyc=?,state=?", model.KYC, model.UserState.Bytes()).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update user kyc upd=%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).Errorf("failed to update user")
	}
	return nil
}

func userSchema(model *observer.User) *UserSchema {
	return &UserSchema{
		Ref:    model.UserRef.Bytes(),
		KYC:    model.KYCStatus,
		Status: model.Status,
		State:  model.State,
		Public: model.Public,
	}
}
