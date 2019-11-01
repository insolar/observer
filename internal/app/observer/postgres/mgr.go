package postgres

import (
	"github.com/go-pg/pg"
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

	PaymentFrequency string `sql:"period,notnull"`

	AmountDue string `sql:"amount,notnull"`
	Status    string `sql:",notnull"`
	State     []byte `sql:",notnull"`
}

type MGRSequence struct {
	tableName struct{} `sql:"mgr_sequence"`
	Index     int
	GroupRef  []byte
	UserRef   []byte
	DrawDate  int64
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
	_, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert(row)

	if err != nil {
		return errors.Wrapf(err, "failed to insert mgr %v, %v", row, err.Error())
	}

	for i, _ := range model.Sequence {
		seq := mgrSequence(i, model)
		_, err := s.db.Model(seq).
			OnConflict("DO NOTHING").
			Insert(seq)
		if err != nil {
			return errors.Wrapf(err, "failed to insert mgr sequence %v, %v", row, err.Error())
		}
	}
	return nil
}

func (s *MGRStorage) Update(model *observer.MGRUpdate) error {
	if model == nil {
		s.log.Warnf("trying to apply nil update mgr model")
		return nil
	}

	var arrRef []string
	for _, v := range model.Sequence {
		arrRef = append(arrRef, v.String())
	}

	res, err := s.db.Model(&MGRSchema{}).
		Where("state=?", model.PrevState.Bytes()).
		Set("period=?,amount=?", model.PaymentFrequency, model.AmountDue).
		Set("sequence=?", pg.Array(arrRef)).
		Set("start_date=?", model.StartRoundDate).
		Set("fin_date=?", model.FinishRoundDate).
		Set("next_payment=?", model.NextPaymentTime).
		Set("state=?", model.MGRState.Bytes()).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update mgr =%v", model)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("upd", model).Errorf("failed to update mgr")
	}
	return nil
}

func mgrSchema(model *observer.MGR) *MGRSchema {
	var arrRef []string
	for _, v := range model.Sequence {
		arrRef = append(arrRef, v.String())
	}
	return &MGRSchema{
		Ref:              model.Ref.Bytes(),
		GroupReference:   model.GroupReference.Bytes(),
		AmountDue:        model.AmountDue,
		PaymentFrequency: model.PaymentFrequency,
		StartRoundDate:   model.StartRoundDate,
		FinishRoundDate:  model.FinishRoundDate,
		NextPaymentDate:  model.NextPaymentTime,
		Status:           model.Status,
		State:            model.State.Bytes(),
	}
}

func mgrSequence(index int, model *observer.MGR) *MGRSequence {
	return &MGRSequence{
		Index:    index + 1,
		GroupRef: model.GroupReference.Bytes(),
		UserRef:  model.Sequence[index].Bytes(),
		// TODO: get correct draw date, wait from insolar
		DrawDate: model.NextPaymentTime,
	}
}
