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
	Active    bool `sql:",notnull"`
}

type MGRSwap struct {
	tableName       struct{} `sql:"swap"`
	GroupRef        []byte   `sql:",notnull"`
	FromPosition    int      `sql:",notnull"`
	FromRef         []byte   `sql:",notnull"`
	ToPosition      int      `sql:",notnull"`
	ToRef           []byte   `sql:",notnull"`
	Timestamp       int64    `sql:",notnull"`
	NotificationRef []byte
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

	_, err := s.db.Model(&MGRSchema{}).
		Where("state=?", model.PrevState.Bytes()).
		Set("period=?,amount=?", model.PaymentFrequency, model.AmountDue).
		Set("start_date=?", model.StartRoundDate).
		Set("fin_date=?", model.FinishRoundDate).
		Set("next_payment=?", model.NextPaymentTime).
		Set("state=?", model.MGRState.Bytes()).
		Update()

	if err != nil {
		return errors.Wrapf(err, "failed to update mgr =%v", model)
	}

	for i, _ := range model.Sequence {
		seq := mgrSequenceUpdate(i, model)
		isInserted, err := s.db.Model(seq).
			Where("group_ref=?", seq.GroupRef).
			Where("user_ref=?", seq.UserRef).
			Set("index=?", seq.Index).
			Set("draw_date=?", seq.DrawDate).
			Set("active=?", seq.Active).
			SelectOrInsert()
		if !isInserted {
			_, err := s.db.Model(&MGRSequence{}).
				Where("group_ref=?", seq.GroupRef).
				Where("user_ref=?", seq.UserRef).
				Set("index=?", seq.Index).
				Set("draw_date=?", seq.DrawDate).
				Set("active=?", seq.Active).
				Update()
			if err != nil {
				return errors.Wrapf(err, "failed to update mgr sequence %v %v", model, err.Error())
			}
		}
		if err != nil {
			return errors.Wrapf(err, "failed to insert mgr sequence %v %v", model, err.Error())
		}
	}

	var indexFrom, indexTo int

	for i, v := range model.Sequence {
		if v.Member == model.SwapProcess.From {
			indexFrom = i + 1
		}
		if v.Member == model.SwapProcess.To {
			indexTo = i + 1
		}
	}

	if !model.SwapProcess.From.IsEmpty() && !model.SwapProcess.From.IsEmpty() {
		swap := mgrSwapUpdate(indexFrom, indexTo, model)
		_, err = s.db.Model(swap).
			Where("group_ref=?", model.GroupReference.Bytes()).
			Where("to_ref=?", model.SwapProcess.To.Bytes()).
			Where("notification_ref IS NULL").
			Set("group_ref=?", model.GroupReference.Bytes()).
			Set("to_ref=?", model.SwapProcess.To.Bytes()).
			Set("from_ref=?", model.SwapProcess.From.Bytes()).
			Set("from_position=?", indexFrom).
			Set("to_position=?", indexTo).
			Set("timestamp=?", model.Timestamp).
			Insert()
		if err != nil {
			return errors.Wrapf(err, "failed to insert mgr swap %v %v", model, err.Error())
		}
	}

	return nil
}

func mgrSchema(model *observer.MGR) *MGRSchema {
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
		UserRef:  model.Sequence[index].Member.Bytes(),
		DrawDate: model.Sequence[index].DueDate,
	}
}

func mgrSequenceUpdate(index int, model *observer.MGRUpdate) *MGRSequence {
	return &MGRSequence{
		Index:    index + 1,
		GroupRef: model.GroupReference.Bytes(),
		UserRef:  model.Sequence[index].Member.Bytes(),
		DrawDate: model.Sequence[index].DueDate,
		Active:   model.Sequence[index].IsActive,
	}
}

func mgrSwapUpdate(indexFrom int, indexTo int, model *observer.MGRUpdate) *MGRSwap {
	return &MGRSwap{
		GroupRef:     model.GroupReference.Bytes(),
		FromRef:      model.SwapProcess.From.Bytes(),
		ToRef:        model.SwapProcess.To.Bytes(),
		FromPosition: indexFrom,
		ToPosition:   indexTo,
		Timestamp:    model.Timestamp,
	}
}
