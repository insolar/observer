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
	"strconv"
)

type UserCollector struct {
	log *logrus.Logger
}

func NewUserCollector(log *logrus.Logger) *UserCollector {
	return &UserCollector{
		log: log,
	}
}

type User struct {
	foundation.BaseContract
	Pulse       insolar.PulseNumber
	Source      string
	MemberRef   insolar.Reference
	KYCStatus   bool
	MemberShips []insolar.Reference
	Key         string
}

func (c *UserCollector) Collect(rec *observer.Record) *observer.User {
	if rec == nil {
		return nil
	}
	actCandidate := observer.CastToActivate(rec)

	if !actCandidate.IsActivate() {
		return nil
	}

	act := actCandidate.Virtual.GetActivate()

	prototypeRef, _ := insolar.NewReferenceFromString("0111A5tDgkPiUrCANU8NTa73b7w6pWGRAUxJTYFXwTnR")
	if !act.Image.Equal(*prototypeRef) {
		return nil
	}

	user, err := c.build(actCandidate)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build user"))
		return nil
	}
	return user
}

func (c *UserCollector) build(act *observer.Activate) (*observer.User, error) {
	if act == nil {
		return nil, errors.New("trying to create user from non complete builder")
	}

	var user User

	err := insolar.Deserialize(act.Virtual.GetActivate().Memory, &user)
	if err != nil {
		return nil, err
	}

	fmt.Println("Insert new user ref:", insolar.NewReference(act.ObjectID).String())
	return &observer.User{
		UserRef:   *insolar.NewReference(act.ObjectID),
		KYCStatus: user.KYCStatus,
		Status:    "SUCCESS",
		State:     act.ID.Bytes(),
		Public:    user.Key,
	}, nil
}

func userKYC(act *observer.Record) (bool, int64, string, error) {
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
		return false, 0, "", errors.New("invalid record to get user memory")
	}

	var user User

	err := insolar.Deserialize(memory, &user)
	if err != nil {
		log.Error(errors.New("failed to deserialize user memory"))
	}

	pn := user.Pulse.String()
	kycDate, err := strconv.ParseInt(pn, 10, 64)

	return user.KYCStatus, kycDate, user.Source, nil
}
