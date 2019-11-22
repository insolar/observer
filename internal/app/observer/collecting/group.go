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
	"reflect"
)

type GroupCollector struct {
	log       *logrus.Logger
	results   observer.ResultCollector
	activates observer.ActivateCollector
	halfChain observer.ChainCollector
	chains    observer.ChainCollector
}

func NewGroupCollector(log *logrus.Logger) *GroupCollector {
	results := NewResultCollector(isGroupCreationCall, successResult)
	activates := NewActivateCollector(isGroupNew, isGroupActivate)
	resultRelation := &RelationDesc{
		Is:     isCoupledResult,
		Origin: coupledResultOrigin,
		Proper: isCoupledResult,
	}
	activateRelation := &RelationDesc{
		Is:     isCoupledActivate,
		Origin: coupledActivateOrigin,
		Proper: isCoupledActivate,
	}
	userCreateGroupCall := &RelationDesc{
		Is: isUserGroupCreateCall,
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: isUserGroupCreateCall,
	}
	userRelation := &RelationDesc{
		Is: func(chain interface{}) bool {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return isUserGroupCreateCall(c.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(c.Parent)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return isUserGroupCreateCall(c.Parent)
		},
	}
	return &GroupCollector{
		results:   results,
		activates: activates,
		halfChain: NewChainCollector(userCreateGroupCall, activateRelation),
		chains:    NewChainCollector(resultRelation, userRelation),
	}
}

func (c *GroupCollector) Collect(rec *observer.Record) *observer.Group {
	res := c.results.Collect(rec)
	act := c.activates.Collect(rec)
	half := c.halfChain.Collect(rec)

	if act != nil {
		half = c.halfChain.Collect(act)
	}

	var chain *observer.Chain
	if res != nil {
		chain = c.chains.Collect(res)
	}

	if half != nil {
		chain = c.chains.Collect(half)
	}

	if chain == nil {
		return nil
	}

	coupleAct, coupleRes, request := c.unwrapGroupChain(chain)

	g, err := c.build(coupleAct, coupleRes, request)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build group"))
		return nil
	}
	return g
}

type Group struct {
	ChairMan    insolar.Reference
	Treasurer   *insolar.Reference
	Title       string
	Membership  foundation.StableMap
	Goal        string
	ProductType *observer.ProductType
	Product     *insolar.Reference
	Balance     *insolar.Reference
	Image       string
	invitedUser int
}

type Membership struct {
	MemberRef    insolar.Reference
	MemberRole   RoleMembership
	MemberStatus StatusMemberShip
	AmountDue    uint64
	JoinPulse    insolar.PulseNumber
	IsAnonymous  bool
}

func (c *GroupCollector) build(act *observer.Activate, res *observer.Result, req *observer.Request) (*observer.Group, error) {
	if res == nil || act == nil {
		return nil, errors.New("trying to create group from non complete builder")
	}

	if res.Virtual.GetResult().Payload == nil {
		return nil, errors.New("group creation result payload is nil")
	}

	activate := act.Virtual.GetActivate()
	state := c.initialGroupState(activate)
	date, err := act.ID.Pulse().AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert group create pulse (%d) to time", act.ID.Pulse())
	}

	logrus.Info("Insert new group ref:", insolar.NewReference(act.ObjectID).String())
	resultGroup := observer.Group{
		Ref:        *insolar.NewReference(act.ObjectID),
		Title:      state.Title,
		Goal:       state.Goal,
		Image:      state.Image,
		ChairMan:   state.ChairMan,
		Membership: state.Membership,
		Status:     "SUCCESS",
		State:      act.ID,
		Timestamp:  date.Unix(),
	}

	if state.Treasurer != nil {
		resultGroup.Treasurer = *state.Treasurer
	}

	if state.ProductType != nil {
		resultGroup.ProductType = *state.ProductType
	}

	if state.Product != nil {
		resultGroup.Product = *state.Product
	}
	return &resultGroup, nil
}

func isUserGroupCreateCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "CreateGroup" {
		return false
	}

	if in.Prototype == nil {
		return false
	}
	prototypeRef, _ := insolar.NewReferenceFromString("0111A5tDgkPiUrCANU8NTa73b7w6pWGRAUxJTYFXwTnR")
	return in.Prototype.Equal(*prototypeRef)
}

func isGroupActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9")
	return act.Image.Equal(*prototypeRef)
}
func isGroupCreationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "group.create" || args.Params.CallSite == "group.initialize"
}
func isGroupNew(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "New" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9")
	return in.Prototype.Equal(*prototypeRef)
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

func (c *GroupCollector) unwrapGroupChain(chain *observer.Chain) (*observer.Activate, *observer.Result, *observer.Request) {

	half := chain.Child.(*observer.Chain)
	coupledAct, ok := half.Child.(*observer.CoupledActivate)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.Chain", reflect.TypeOf(chain.Child)))
		return nil, nil, nil
	}
	if coupledAct.Activate == nil {
		log.Error(errors.New("invalid coupled activate chain, child is nil"))
		return nil, nil, nil
	}
	actRecord := coupledAct.Activate

	coupledRes, ok := chain.Parent.(*observer.CoupledResult)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.Chain", reflect.TypeOf(chain.Parent)))
		return nil, nil, nil
	}
	if coupledRes.Result == nil {
		log.Error(errors.New("invalid coupled result chain, child is nil"))
		return nil, nil, nil
	}
	resRecord := coupledRes.Result
	reqRecord := coupledRes.Request

	return actRecord, resRecord, reqRecord
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
