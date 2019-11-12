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

type NotificationSchema struct {
	tableName struct{} `sql:"notifications"`

	Ref            []byte `sql:",pk"`
	UserReference  []byte `sql:"user_ref,notnull"`
	GroupReference []byte `sql:"group_ref,notnull"`
	Type           string `sql:"type,notnull"`
	Timestamp      int64  `sql:"timestamp,notnull"`
}

func NewNotificationStorage(obs *observability.Observability, db orm.DB) *NotificationStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_notification_storage_error_counter",
		Help: "",
	})
	return &NotificationStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

type NotificationStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func (s *NotificationStorage) Insert(model *observer.Notification) error {
	if model == nil {
		s.log.Warnf("trying to insert nil notification model")
		return nil
	}
	row := notificationSchema(model)

	_, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert notification %v, %v", row, err.Error())
	}

	_, err = s.db.Model(&MGRSwap{}).
		Where("group_ref=?", model.GroupReference.Bytes()).
		Where("to_ref=?", model.UserReference.Bytes()).
		Where("notification_ref IS NULL").
		Set("notification_ref=?", model.Ref.Bytes()).
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update swap %v, %v", row, err.Error())
	}

	return nil
}

func notificationSchema(model *observer.Notification) *NotificationSchema {
	var notificationType string
	switch model.Type {
	case observer.NotificationInvite:
		notificationType = "invite"
	case observer.NotificationContribution:
		notificationType = "contribute"
	case observer.NotificationDeactivate:
		notificationType = "deactivate"
	case observer.NotificationFinishMGRRound:
		notificationType = "finishMgrRound"
	case observer.NotificationSwap:
		notificationType = "swap"
	}

	return &NotificationSchema{
		Ref:            model.Ref.Bytes(),
		UserReference:  model.UserReference.Bytes(),
		GroupReference: model.GroupReference.Bytes(),
		Type:           notificationType,
		Timestamp:      model.Timestamp,
	}
}
