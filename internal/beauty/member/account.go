package member

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/account"
	proxyAccount "github.com/insolar/insolar/logicrunner/builtin/proxy/account"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func isAccountActivate(act *record.Activate) bool {
	return act.Image.Equal(*proxyAccount.PrototypeReference)
}

func isNewAccount(rec *record.Material) bool {
	_, ok := rec.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		return false
	}
	in := rec.Virtual.GetIncomingRequest()
	return in.Method == "New" && in.Prototype.Equal(*proxyAccount.PrototypeReference)
}

func isAccountAmend(amd *record.Amend) bool {
	return amd.Image.Equal(*proxyAccount.PrototypeReference)
}

func accountBalance(rec *record.Material) string {
	memory := []byte{}
	balance := ""
	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get account memory"))
	}
	acc := account.Account{}
	if err := insolar.Deserialize(memory, &acc); err != nil {
		log.Error(errors.New("failed to deserialize account memory"))
	} else {
		balance = acc.Balance
	}
	return balance
}
