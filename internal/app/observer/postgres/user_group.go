package postgres

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
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
	for _, u := range model.Members {
		// regular roles
		row := userGroupMemberSchema(model, u, 2)
		err := s.insertRow(row)
		if err != nil {
			return err
		}
	}
	// chairmen or creator
	row := userGroupMemberSchema(model, model.ChairMan, 1)
	return s.insertRow(row)
}

func (s *UserGroupStorage) insertRow(row *UserGroupSchema) error {
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

func userGroupMemberSchema(group *observer.Group, userRef insolar.Reference, role int) *UserGroupSchema {
	return &UserGroupSchema{
		UserRef:  userRef.Bytes(),
		GroupRef: group.Ref.Bytes(),
		Role:     role,
	}
}
