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

type GroupSchema struct {
	tableName struct{} `sql:"groups"`

	Ref            []byte `sql:",pk"`
	Title          string
	Goal           string
	Purpose        string
	GroupOwner     []byte
	TreasureHolder []byte
	Status         string
}

func NewGroupStorage(obs *observability.Observability, db orm.DB) *GroupStorage {
	errorCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "observer_group_storage_error_counter",
		Help: "",
	})
	obs.Metrics().MustRegister(errorCounter)
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

func groupSchema(model *observer.Group) *GroupSchema {
	return &GroupSchema{
		Ref:        model.Ref.Bytes(),
		Title:      model.Title,
		Goal:       model.Goal,
		GroupOwner: model.ChairMan.Bytes(),
		Purpose:    model.Purpose,
		Status:     model.Status,
	}
}
