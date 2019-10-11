package postgres

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/configuration"
	observer2 "github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"time"
)

type UserGroupSchema struct {
	tableName struct{} `sql:"user_group"`

	UserRef         []byte
	GroupRef        []byte
	Role            string
	Status          string
	StatusTimestamp int64
}

func NewUserGroupStorage(obs *observability.Observability, db orm.DB) *UserGroupStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_user_group_storage_error_counter",
		Help: "",
	})
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

func (s *UserGroupStorage) Insert(model *observer2.Group) error {
	if model == nil {
		s.log.Warnf("trying to insert nil user-group model")
		return nil
	}
	// User status
	// 1	invited
	// 2	active
	// 3	rejected
	// 4	expelled
	for _, u := range model.Members {
		// regular roles with invited status
		row := userGroupMemberSchema(model, u, "member", "invited")
		err := s.insertRow(row)
		if err != nil {
			return err
		}
	}
	// chairmen or creator with active status
	row := userGroupMemberSchema(model, model.ChairMan, "chairman", "active")
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

func userGroupMemberSchema(group *observer2.Group, userRef insolar.Reference, role string, status string) *UserGroupSchema {
	return &UserGroupSchema{
		UserRef:         userRef.Bytes(),
		GroupRef:        group.Ref.Bytes(),
		Role:            role,
		Status:          status,
		StatusTimestamp: time.Now().Unix(),
	}
}
