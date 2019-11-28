package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type MGRCollector struct {
	log *logrus.Logger
}

func NewMGRCollector(log *logrus.Logger) *MGRCollector {
	return &MGRCollector{
		log: log,
	}
}

type MerryGoRound struct {
	GroupReference  *insolar.Reference
	StartRoundDate  int64      // unix timestamp
	FinishRoundDate int64      // unix timestamp
	AmountDue       string     // amount of money
	Sequence        []Sequence // array of users refs, [0] element is first in queue
	SwapProcess     Swap       // Swap started and finished processes
}

type Sequence struct {
	Member   insolar.Reference
	DrawDate int64
	IsActive bool
}

type Swap struct {
	From   insolar.Reference // User who initialized swap process
	To     insolar.Reference // User who will accept request
	Status StatusSwap
}

// Status type of swap
type StatusSwap int

const (
	SwapNotInit StatusSwap = iota
	SwapPropose
)

func (c *MGRCollector) Collect(rec *observer.Record) *observer.MGR {
	if rec == nil {
		return nil
	}
	actCandidate := observer.CastToActivate(rec)

	if !actCandidate.IsActivate() {
		return nil
	}

	act := actCandidate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A6L4ytii4Z9jWLJpFqjDkH8ZRZ8HNscmmzsBF85i")
	if !act.Image.Equal(*prototypeRef) {
		return nil
	}

	mgr, err := c.build(actCandidate)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build mgr"))
		return nil
	}
	return mgr
}

func (c *MGRCollector) build(act *observer.Activate) (*observer.MGR, error) {
	if act == nil {
		return nil, errors.New("trying to create mgr from non complete builder")
	}

	var mgr MerryGoRound

	err := insolar.Deserialize(act.Virtual.GetActivate().Memory, &mgr)
	if err != nil {
		return nil, err
	}

	logrus.Info("Insert new product ref:", insolar.NewReference(act.ObjectID).String())
	var seq []observer.Sequence

	for _, v := range mgr.Sequence {
		seq = append(seq, observer.Sequence{Member: v.Member, DrawDate: v.DrawDate, IsActive: v.IsActive})
	}

	resultProduct := observer.MGR{
		Ref:             *insolar.NewReference(act.ObjectID),
		StartRoundDate:  mgr.StartRoundDate,
		FinishRoundDate: mgr.FinishRoundDate,
		AmountDue:       mgr.AmountDue,
		Sequence:        seq,
		Status:          "SUCCESS",
		State:           act.ID,
	}

	if mgr.GroupReference != nil {
		resultProduct.GroupReference = *mgr.GroupReference
	}

	return &resultProduct, nil
}

func mgrUpdate(act *observer.Record) (*MerryGoRound, error) {
	var memory []byte
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get mgr memory"))
	}

	if memory == nil {
		log.Warn(errors.New("group memory is nil"))
		return nil, errors.New("invalid record to get mgr memory")
	}

	var mgr MerryGoRound

	err := insolar.Deserialize(memory, &mgr)
	if err != nil {
		log.Error(errors.New("failed to deserialize mgr memory"))
	}

	return &mgr, nil
}
