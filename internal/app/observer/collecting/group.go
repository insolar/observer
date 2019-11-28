package collecting

import (
	"encoding/json"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type GroupCollector struct {
	log *logrus.Logger
}

func NewGroupCollector(log *logrus.Logger) *GroupCollector {
	return &GroupCollector{
		log: log,
	}
}

func (c *GroupCollector) Collect(rec *observer.Record) *observer.Group {
	if rec == nil {
		return nil
	}
	actCandidate := observer.CastToActivate(rec)

	if !actCandidate.IsActivate() {
		return nil
	}

	act := actCandidate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9")
	if !act.Image.Equal(*prototypeRef) {
		return nil
	}

	g, err := c.build(actCandidate)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build group"))
		return nil
	}
	return g
}

type Group struct {
	ChairMan         insolar.Reference
	Treasurer        *insolar.Reference
	Title            string
	Membership       foundation.StableMap
	Goal             string
	ProductType      *observer.ProductType
	StartDate        int64
	PaymentFrequency PaymentFrequency
	Product          *insolar.Reference
	Balance          *insolar.Reference
	Image            string
	invitedUser      int
}

type Membership struct {
	MemberRef    insolar.Reference
	MemberRole   RoleMembership
	MemberStatus StatusMemberShip
	MemberGoal   uint64
	JoinPulse    insolar.PulseNumber
	IsAnonymous  bool
}

func (c *GroupCollector) build(act *observer.Activate) (*observer.Group, error) {
	if act == nil {
		return nil, errors.New("trying to create group from non complete builder")
	}

	activate := act.Virtual.GetActivate()
	state := c.initialGroupState(activate)
	date, err := act.ID.Pulse().AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert group create pulse (%d) to time", act.ID.Pulse())
	}

	logrus.Info("Insert new group ref:", insolar.NewReference(act.ObjectID).String())
	resultGroup := observer.Group{
		Ref:              *insolar.NewReference(act.ObjectID),
		Title:            state.Title,
		Goal:             state.Goal,
		Image:            state.Image,
		ChairMan:         state.ChairMan,
		Membership:       state.Membership,
		Status:           "SUCCESS",
		State:            act.ID,
		PaymentFrequency: state.PaymentFrequency.String(),
		Timestamp:        date.Unix(),
	}

	if state.Treasurer != nil {
		resultGroup.Treasurer = *state.Treasurer
	}

	if state.ProductType != nil {
		resultGroup.ProductType = *state.ProductType
	}

	if state.Product != nil {
		resultGroup.ProductRef = *state.Product
	}
	return &resultGroup, nil
}

func (c *GroupCollector) initialGroupState(act *record.Activate) *Group {
	g := Group{}
	err := insolar.Deserialize(act.Memory, &g)
	if err != nil {
		log.Error(errors.New("failed to deserialize group contract state"))
	}
	if g.Membership != nil {
		for _, v := range g.Membership {
			byt := []byte(v)
			var membership Membership
			if err := json.Unmarshal(byt, &membership); err != nil {
				return nil
			}
		}

	}
	return &g
}

func groupUpdate(act *observer.Record) (*Group, error) {
	var memory []byte
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get group memory"))
	}

	if memory == nil {
		log.Warn(errors.New("group memory is nil"))
		return nil, errors.New("invalid record to get group memory")
	}

	var group Group

	err := insolar.Deserialize(memory, &group)
	if err != nil {
		log.Error(errors.New("failed to deserialize group memory"))
	}

	return &group, nil
}
