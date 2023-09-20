package collecting

import (
	"errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/mainnet/application/builtin/contract/account"
	proxyAccount "github.com/insolar/mainnet/application/builtin/proxy/account"

	"github.com/insolar/observer/internal/app/observer"
)

type BalanceCollector struct {
	log insolar.Logger
}

func NewBalanceCollector(log insolar.Logger) *BalanceCollector {
	return &BalanceCollector{
		log: log,
	}
}

func (c *BalanceCollector) Collect(rec *observer.Record) *observer.Balance {
	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isAccountAmend(v.Amend) {
		return nil
	}
	amd := rec.Virtual.GetAmend()
	balance := balance(rec)
	return &observer.Balance{
		PrevState:    amd.PrevState,
		AccountState: rec.ID,
		Balance:      balance,
	}
}

func isAccountAmend(amd *record.Amend) bool {
	return amd.Image.Equal(*proxyAccount.PrototypeReference)
}

func balance(act *observer.Record) string {
	var memory []byte
	balance := ""
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get account memory"))
	}

	if memory == nil {
		log.Warn(errors.New("account memory is nil"))
		return "0"
	}

	acc := account.Account{}
	if err := insolar.Deserialize(memory, &acc); err != nil {
		log.Error(errors.New("failed to deserialize account memory"))
	} else {
		balance = acc.Balance
	}
	return balance
}
