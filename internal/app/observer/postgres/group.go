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

type GroupSchema struct {
	tableName struct{} `sql:"groups"`

	Ref            []byte `sql:",pk"`
	Title          string
	Goal           string
	Purpose        string
	GroupOwner     []byte
	TreasureHolder []byte
	Status         string
	State          []byte
}

func NewGroupStorage(obs *observability.Observability, db orm.DB) *GroupStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_group_storage_error_counter",
		Help: "",
	})
	return &GroupStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

type GroupStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func (s *GroupStorage) Insert(model *observer.Group) error {
	if model == nil {
		s.log.Warnf("trying to insert nil group model")
		return nil
	}
	row := groupSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert group %v, %v", row, err.Error())
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("group_row", row).
			Errorf("failed to insert group")
	}
	return nil
}

func (s *GroupStorage) Update(model *observer.GroupUpdate) error {
	if model == nil {
		s.log.Warnf("trying to apply nil update group model")
		return nil
	}

	res, err := s.db.Model(&GroupSchema{}).
		Where("state=?", model.PrevState.Bytes()).
		Set("purpose=?,goal=?", model.Purpose, model.Goal).
		Set("type=?", model.ProductType).
		Set("treasure_holder=?", model.Treasurer.Bytes()).
		Set("state=?", model.GroupState.Bytes()).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update group =%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).Errorf("failed to update group")
	}
	if !model.Treasurer.IsEmpty() {
		res, err = s.db.Model(&UserGroupSchema{}).
			Where("user_ref=?", model.Treasurer.Bytes()).
			Set("role=?", "treasurer").
			Update()

		if err != nil {
			return errors.Wrapf(err, "failed to update user_group =%v", model)
		}

		if res.RowsAffected() == 0 {
			s.errorCounter.Inc()
			s.log.WithField("upd", model).Errorf("failed to update user_group")
		}
	}
	if model.Membership != nil {
		for _, membershipStr := range model.Membership {
			byt := []byte(membershipStr)
			var membership collecting.Membership
			if err := json.Unmarshal(byt, &membership); err != nil {
				return nil
			}
			var dbState string
			var dbRole string
			switch membership.MemberStatus {
			case collecting.StatusInvite:
				dbState = "invited"
			case collecting.StatusActive:
				dbState = "active"
			case collecting.StatusReject:
				dbState = "rejected"
			}

			switch membership.MemberRole {
			case collecting.RoleChairMan:
				dbRole = "admin"
			case collecting.RoleTreasure:
				dbRole = "treasurer"
			case collecting.RoleMember:
				dbRole = "member"
			}

			count, err := s.db.Model(&UserGroupSchema{}).Where("group_ref=?", model.GroupReference.Bytes()).
				Where("user_ref=?", membership.MemberRef.Bytes()).Count()
			if count == 0 {

				row := &UserGroupSchema{
					UserRef:         membership.MemberRef.Bytes(),
					GroupRef:        model.GroupReference.Bytes(),
					Role:            "member",
					Status:          "invited",
					StatusTimestamp: model.Timestamp,
				}

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
			}

			res, err = s.db.Model(&UserGroupSchema{}).
				Where("group_ref=?", model.GroupReference.Bytes()).
				Where("user_ref=?", membership.MemberRef.Bytes()).
				Set("status=?", dbState).
				Set("role=?", dbRole).
				Update()

			if err != nil {
				return errors.Wrapf(err, "failed to update user_group =%v", model)
			}

			if res.RowsAffected() == 0 {
				s.errorCounter.Inc()
				s.log.WithField("upd", model).Errorf("failed to update user_group")
			}
		}

	}
	return nil
}

func groupSchema(model *observer.Group) *GroupSchema {
	return &GroupSchema{
		Ref:        model.Ref.Bytes(),
		Title:      model.Title,
		Goal:       model.Goal,
		GroupOwner: model.ChairMan.Bytes(),
		Purpose:    model.Purpose,
		Status:     model.Status,
		State:      model.State.Bytes(),
	}
}
