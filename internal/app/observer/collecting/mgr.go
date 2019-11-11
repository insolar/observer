package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"reflect"
)

type MGRCollector struct {
	log       *logrus.Logger
	results   observer.ResultCollector
	activates observer.ActivateCollector
	halfChain observer.ChainCollector
	chains    observer.ChainCollector
}

func NewMGRCollector(log *logrus.Logger) *MGRCollector {
	results := NewResultCollector(isMGRCreationCall, successResult)
	activates := NewActivateCollector(isMGRNew, isMGRActivate)
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
		Is: isGroupMGRCreateCall,
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: isGroupMGRCreateCall,
	}
	userRelation := &RelationDesc{
		Is: func(chain interface{}) bool {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return isGroupMGRCreateCall(c.Parent)
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
			return isGroupMGRCreateCall(c.Parent)
		},
	}
	return &MGRCollector{
		results:   results,
		activates: activates,
		halfChain: NewChainCollector(userCreateGroupCall, activateRelation),
		chains:    NewChainCollector(resultRelation, userRelation),
	}
}

type MerryGoRound struct {
	foundation.BaseContract
	GroupReference   insolar.Reference
	StartRoundDate   int64      // unix timestamp
	FinishRoundDate  int64      // unix timestamp
	AmountDue        string     // amount of money
	PaymentFrequency string     // daily, weekly, monthly
	NextPaymentTime  int64      // unix timestamp, need to be calculated
	Sequence         []Sequence // array of users refs, [0] element is first in queue
	SwapProcess      Swap       // Swap started and finished processes
}

type Sequence struct {
	Member   insolar.Reference
	DueDate  int64
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

	coupleAct, coupleRes, request := c.unwrapMGRChain(chain)

	g, err := c.build(coupleAct, coupleRes, request)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build group"))
		return nil
	}
	return g
}

func (c *MGRCollector) build(act *observer.Activate, res *observer.Result, req *observer.Request) (*observer.MGR, error) {
	if res == nil || act == nil {
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
		seq = append(seq, observer.Sequence{Member: v.Member, DueDate: v.DueDate, IsActive: v.IsActive})
	}
	return &observer.MGR{
		Ref:              *insolar.NewReference(act.ObjectID),
		GroupReference:   mgr.GroupReference,
		StartRoundDate:   mgr.StartRoundDate,
		FinishRoundDate:  mgr.FinishRoundDate,
		AmountDue:        mgr.AmountDue,
		PaymentFrequency: mgr.PaymentFrequency,
		NextPaymentTime:  mgr.NextPaymentTime,
		Sequence:         seq,
		Status:           "SUCCESS",
		State:            *insolar.NewReference(act.ID),
	}, nil
}

func isMGRCreationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}
	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "group.setMGR"
}

func isMGRActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A6L4ytii4Z9jWLJpFqjDkH8ZRZ8HNscmmzsBF85i")
	return act.Image.Equal(*prototypeRef)
}

func isMGRNew(chain interface{}) bool {
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
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A6L4ytii4Z9jWLJpFqjDkH8ZRZ8HNscmmzsBF85i")
	return in.Prototype.Equal(*prototypeRef)
}

func isGroupMGRCreateCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "CreateMGRProduct" {
		return false
	}

	if in.Prototype == nil {
		return false
	}
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9")
	return in.Prototype.Equal(*prototypeRef)
}

func (c *MGRCollector) unwrapMGRChain(chain *observer.Chain) (*observer.Activate, *observer.Result, *observer.Request) {

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
