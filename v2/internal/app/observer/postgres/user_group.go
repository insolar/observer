package postgres

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/observability"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type UserGroupSchema struct {
	tableName struct{} `sql:"user_group"`

	UserRef  []byte
	GroupRef []byte
	Role     int
}

func NewUserGroupStorage(obs *observability.Observability, db orm.DB) *UserGroupStorage {
	errorCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "observer_user_group_storage_error_counter",
		Help: "",
	})
	obs.Metrics().MustRegister(errorCounter)
	return &UserGroupStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

type UserGroupStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func (s *UserGroupStorage) Insert(model *observer.Group) error {
	if model == nil {
		s.log.Warnf("trying to insert nil user-group model")
		return nil
	}
	row := userGroupSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert user-group %v, %v", row, err.Error())
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("user-group_row", row).
			Errorf("failed to insert user-group")
	}
	return nil
}

func userGroupSchema(model *observer.Group) *UserGroupSchema {
	return &UserGroupSchema{
		UserRef:  model.ChairMan,
		GroupRef: model.Ref.Bytes(),
		Role:     1,
	}
}
