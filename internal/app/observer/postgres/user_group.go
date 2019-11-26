package postgres

import (
	"encoding/json"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/collecting"
	"github.com/insolar/observer/observability"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type UserGroupSchema struct {
	tableName struct{} `sql:"user_group"`

	UserRef         []byte
	GroupRef        []byte
	Role            string
	Status          string
	AmountDue       uint64
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

func (s *UserGroupStorage) Insert(group *observer.Group) error {
	if group == nil {
		s.log.Warnf("trying to insert nil user-group group")
		return nil
	}

	if group.Membership != nil {
		for _, membershipStr := range group.Membership {
			byt := []byte(membershipStr)
			var membership collecting.Membership
			if err := json.Unmarshal(byt, &membership); err != nil {
				return nil
			}
			row := userGroupMemberSchema(group, membership)
			err := s.insertRow(row)
			if err != nil {
				return err
			}
		}
	}
	return nil
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

func userGroupMemberSchema(group *observer.Group, membership collecting.Membership) *UserGroupSchema {
	return &UserGroupSchema{
		UserRef:         membership.MemberRef.Bytes(),
		GroupRef:        group.Ref.Bytes(),
		Role:            membership.MemberRole.String(),
		Status:          membership.MemberStatus.String(),
		AmountDue:       membership.MemberGoal,
		StatusTimestamp: group.Timestamp,
	}
}
