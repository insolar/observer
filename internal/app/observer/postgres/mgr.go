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

type MGRSchema struct {
	tableName      struct{} `sql:"product_mgr"`
	Ref            []byte   `sql:",pk"`
	GroupReference []byte   `sql:"group_ref,pk"`

	StartRoundDate  int64 `sql:"start_date,notnull"`
	FinishRoundDate int64 `sql:"fin_date,notnull"`
	NextPaymentDate int64 `sql:"next_payment,notnull"`

	PaymentFrequency string   `sql:"period,notnull"`
	Sequence         [][]byte `sql:",array"`

	AmountDue string `sql:"amount,notnull"`
	Status    string `sql:",notnull"`
	State     []byte `sql:",notnull"`
}

func NewMGRStorage(obs *observability.Observability, db orm.DB) *MGRStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_user_storage_error_counter",
		Help: "",
	})
	return &MGRStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

type MGRStorage struct {
	cfg          *configuration.Configuration
	log          *logrus.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func (s *MGRStorage) Insert(model *observer.MGR) error {
	if model == nil {
		s.log.Warnf("trying to insert nil mgr model")
		return nil
	}
	row := mgrSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert mgr %v, %v", row, err.Error())
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("mgr_row", row).
			Errorf("failed to insert user")
	}
	return nil
}

func (s *MGRStorage) Update(model *observer.MGRUpdate) error {
	if model == nil {
		s.log.Warnf("trying to apply nil update mgr model")
		return nil
	}
	logrus.Info("Not implemented update for mgr:", model)
	return nil
}

func mgrSchema(model *observer.MGR) *MGRSchema {
	var arrRef [][]byte
	for _, v := range model.Sequence {
		arrRef = append(arrRef, v.Bytes())
	}
	return &MGRSchema{
		Ref:              model.Ref.Bytes(),
		GroupReference:   model.GroupReference.Bytes(),
		AmountDue:        model.AmountDue,
		PaymentFrequency: model.PaymentFrequency,
		StartRoundDate:   model.StartRoundDate,
		FinishRoundDate:  model.FinishRoundDate,
		NextPaymentDate:  model.NextPaymentTime,
		Sequence:         arrRef,
		Status:           model.Status,
		State:            model.State,
	}
}
