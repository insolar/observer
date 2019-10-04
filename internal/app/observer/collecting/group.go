package collecting

import (
	"fmt"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	observer2 "github.com/insolar/observer/internal/app/observer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"reflect"
)

type GroupCollector struct {
	log       *logrus.Logger
	results   observer2.ResultCollector
	activates observer2.ActivateCollector
	halfChain observer2.ChainCollector
	chains    observer2.ChainCollector
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
			request := observer2.CastToRequest(chain)
			return request.ID
		},
		Proper: isUserGroupCreateCall,
	}
	userRelation := &RelationDesc{
		Is: func(chain interface{}) bool {
			c, ok := chain.(*observer2.Chain)
			if !ok {
				return false
			}
			return isUserGroupCreateCall(c.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			c, ok := chain.(*observer2.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer2.CastToRequest(c.Parent)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			c, ok := chain.(*observer2.Chain)
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

type Group struct {
	foundation.BaseContract
	ChairMan   insolar.Reference
	Title      string
	Goal       string
	Purpose    string
	Membership []insolar.Reference
	Members    []insolar.Reference
}

func (c *GroupCollector) Collect(rec *observer2.Record) *observer2.Group {
	res := c.results.Collect(rec)
	act := c.activates.Collect(rec)
	half := c.halfChain.Collect(rec)

	if act != nil {
		half = c.halfChain.Collect(act)
	}

	var chain *observer2.Chain
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

type Members struct {
	Members []insolar.Reference
}

func (c *GroupCollector) build(act *observer2.Activate, res *observer2.Result, req *observer2.Request) (*observer2.Group, error) {
	if res == nil || act == nil {
		return nil, errors.New("trying to create group from non complete builder")
	}

	if res.Virtual.GetResult().Payload == nil {
		return nil, errors.New("group creation result payload is nil")
	}
	response := &CreateResponse{}
	res.ParseFirstPayloadValue(response)

	ref, err := insolar.NewReferenceFromBase58(response.Reference)
	if err != nil || ref == nil {
		return nil, errors.New("invalid group reference")
	}

	activate := act.Virtual.GetActivate()
	state := c.initialGroupState(activate)

	members := &Members{}

	req.ParseMemberContractCallParams(members)

	fmt.Println("Insert new group ref:", ref.String())
	return &observer2.Group{
		Ref:        *ref,
		Title:      state.Title,
		ChairMan:   state.ChairMan,
		Goal:       state.Goal,
		Purpose:    state.Purpose,
		Membership: state.Membership,
		Members:    members.Members,
		Status:     "SUCCESS",
	}, nil
}

func isUserGroupCreateCall(chain interface{}) bool {
	request := observer2.CastToRequest(chain)
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
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5tDgkPiUrCANU8NTa73b7w6pWGRAUxJTYFXwTnR")
	return in.Prototype.Equal(*prototypeRef)
}
func isGroupActivate(chain interface{}) bool {
	activate := observer2.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9")
	return act.Image.Equal(*prototypeRef)
}
func isGroupCreationCall(chain interface{}) bool {
	request := observer2.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "group.create"
}
func isGroupNew(chain interface{}) bool {
	request := observer2.CastToRequest(chain)
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
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9")
	return in.Prototype.Equal(*prototypeRef)
}

func (c *GroupCollector) initialGroupState(act *record.Activate) *Group {
	g := Group{}
	err := insolar.Deserialize(act.Memory, &g)
	if err != nil {
		log.Error(errors.New("failed to deserialize group contract state"))
	}
	return &g
}

func (c *GroupCollector) unwrapGroupChain(chain *observer2.Chain) (*observer2.Activate, *observer2.Result, *observer2.Request) {

	half := chain.Child.(*observer2.Chain)
	coupledAct, ok := half.Child.(*observer2.CoupledActivate)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.Chain", reflect.TypeOf(chain.Child)))
		return nil, nil, nil
	}
	if coupledAct.Activate == nil {
		log.Error(errors.New("invalid coupled activate chain, child is nil"))
		return nil, nil, nil
	}
	actRecord := coupledAct.Activate

	coupledRes, ok := chain.Parent.(*observer2.CoupledResult)
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
