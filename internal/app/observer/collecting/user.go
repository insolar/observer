package collecting

import (
	"fmt"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type UserCollector struct {
	log       *logrus.Logger
	collector *BoundCollector
}

func NewUserCollector(log *logrus.Logger) *UserCollector {
	collector := NewBoundCollector(isUserCreationCall, successResult, isUserNew, isUserActivate)
	return &UserCollector{
		collector: collector,
	}
}

type User struct {
	foundation.BaseContract
	MemberRef   insolar.Reference
	KYCStatus   bool
	MemberShips []insolar.Reference
}

type CreateResponse struct {
	Reference string `json:"reference"`
}

func (c *UserCollector) Collect(rec *observer.Record) *observer.User {
	if rec == nil {
		return nil
	}
	couple := c.collector.Collect(rec)
	if couple == nil {
		return nil
	}

	m, err := c.build(couple.Activate, couple.Result)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build user"))
		return nil
	}
	return m
}

func (c *UserCollector) build(act *observer.Activate, res *observer.Result) (*observer.User, error) {
	if res == nil || act == nil {
		return nil, errors.New("trying to create user from non complete builder")
	}

	if res.Virtual.GetResult().Payload == nil {
		return nil, errors.New("user creation result payload is nil")
	}
	response := &CreateResponse{}
	res.ParseFirstPayloadValue(response)

	ref, err := insolar.NewReferenceFromBase58(response.Reference)
	if err != nil || ref == nil {
		return nil, errors.New("invalid user reference")
	}
	var user User

	err = insolar.Deserialize(act.Virtual.GetActivate().Memory, &user)
	if err != nil {
		return nil, err
	}

	fmt.Println("Insert new user ref:", ref.String())
	return &observer.User{
		UserRef:   *ref,
		KYCStatus: user.KYCStatus,
		Status:    "SUCCESS",
		State:     act.ID.Bytes(),
	}, nil
}

func isUserCreationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}
	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "user.create"
}

func isUserActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5tDgkPiUrCANU8NTa73b7w6pWGRAUxJTYFXwTnR")
	return act.Image.Equal(*prototypeRef)
}

func isUserNew(chain interface{}) bool {
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
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5tDgkPiUrCANU8NTa73b7w6pWGRAUxJTYFXwTnR")
	return in.Prototype.Equal(*prototypeRef)
}

func userKYC(act *observer.Record) bool {
	var memory []byte
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get user memory"))
	}

	if memory == nil {
		log.Warn(errors.New("user memory is nil"))
		return false
	}

	var user User

	err := insolar.Deserialize(memory, &user)
	if err != nil {
		log.Error(errors.New("failed to deserialize user memory"))
	}
	return user.KYCStatus
}
