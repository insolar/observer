package postgres

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/observer/configuration"
	observer2 "github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type UserSchema struct {
	tableName struct{} `sql:"users"`

	Ref    []byte `sql:",pk"`
	KYC    bool   `sql:",notnull"`
	Status string `sql:",notnull"`
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

func (s *UserStorage) Insert(model *observer2.User) error {
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

func userSchema(model *observer2.User) *UserSchema {
	return &UserSchema{
		Ref:    model.UserRef.Bytes(),
		KYC:    model.KYCStatus,
		Status: model.Status,
	}
}
