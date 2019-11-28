package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strconv"
)

type SavingCollector struct {
	log *logrus.Logger
}

type NormalSavings struct {
	NSContribute    foundation.StableMap
	StartRoundDate  int64
	NextPaymentDate int64
}

func NewSavingCollector(log *logrus.Logger) *SavingCollector {
	return &SavingCollector{
		log: log,
	}
}

func (c *SavingCollector) Collect(rec *observer.Record) *observer.NormalSaving {
	if rec == nil {
		return nil
	}
	actCandidate := observer.CastToActivate(rec)

	if !actCandidate.IsActivate() {
		return nil
	}

	act := actCandidate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A6Uo4DN71b7FVUjW6yZvPTm4JVk1rYdBpajUUig2") // savings
	if !act.Image.Equal(*prototypeRef) {
		return nil
	}

	saving, err := c.build(actCandidate)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build saving"))
		return nil
	}
	return saving
}

func (c *SavingCollector) build(act *observer.Activate) (*observer.NormalSaving, error) {
	if act == nil {
		return nil, errors.New("trying to create mgr from non complete builder")
	}

	var savings NormalSavings

	err := insolar.Deserialize(act.Virtual.GetActivate().Memory, &savings)
	if err != nil {
		return nil, err
	}

	contMap := make(map[insolar.Reference]int64)

	for userRefStr, contributeDate := range savings.NSContribute {
		userRef, err := insolar.NewReferenceFromString(userRefStr)
		if err != nil {
			return nil, err
		}
		contDate, err := strconv.ParseInt(contributeDate, 10, 64)
		if err != nil {
			return nil, err
		}
		contMap[*userRef] = contDate
	}

	logrus.Info("Insert new saving product ref:", insolar.NewReference(act.ObjectID).String())
	resultProduct := observer.NormalSaving{
		Reference:       *insolar.NewReference(act.ObjectID),
		StartRoundDate:  savings.StartRoundDate,
		NSContribute:    contMap,
		NextPaymentDate: savings.NextPaymentDate,
		State:           act.ID,
	}
	return &resultProduct, nil
}

func savingUpdate(act *observer.Record) (*NormalSavings, error) {
	var memory []byte
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get ns memory"))
	}

	if memory == nil {
		log.Warn(errors.New("group memory is nil"))
		return nil, errors.New("invalid record to get ns memory")
	}

	var ns NormalSavings

	err := insolar.Deserialize(memory, &ns)
	if err != nil {
		log.Error(errors.New("failed to deserialize ns memory"))
	}

	return &ns, nil
}
