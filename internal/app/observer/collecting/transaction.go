/*
 *
 *  Copyright  2019. Insolar Technologies GmbH
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and
 *  limitations under the License.
 */

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
	"reflect"
	"strconv"
)

type TransactionCollector struct {
	log         *logrus.Logger
	collector   *ResultCollector
	balance     *ChainCollector
	transaction *ChainCollector
	activate    *ActivateCollector
}

func NewTransactionCollector(log *logrus.Logger) *TransactionCollector {
	collector := NewResultCollector(isTransactionCreationCall, successResult)
	activate := NewActivateCollector(isTransactionNew, isTransactionActivate)

	balance := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			res, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isTransactionCreationCall(res.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			res, ok := chain.(*observer.CoupledResult)
			if !ok {
				return insolar.ID{}
			}
			return res.Request.ID
		},
		Proper: func(chain interface{}) bool {
			res, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isTransactionCreationCall(res.Request)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			request := observer.CastToRequest(chain)
			return request.IsIncoming()
		},
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			request := observer.CastToRequest(chain)
			return isCreateTransaction(request)
		},
	})
	transaction := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			che, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			res, ok := che.Parent.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isTransactionCreationCall(res.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			che, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			rec, ok := che.Child.(*observer.Record)
			if !ok {
				return insolar.ID{}
			}
			return rec.ID
		},
		Proper: func(chain interface{}) bool {
			che, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			res, ok := che.Parent.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isTransactionCreationCall(res.Request)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			act, ok := chain.(*observer.CoupledActivate)
			if !ok {
				return false
			}
			return isTransactionNew(act.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			act, ok := chain.(*observer.CoupledActivate)
			if !ok {
				return insolar.ID{}
			}
			return act.Request.Reason()
		},
		Proper: func(chain interface{}) bool {
			act, ok := chain.(*observer.CoupledActivate)
			if !ok {
				return false
			}
			return isTransactionNew(act.Request)
		},
	})
	return &TransactionCollector{
		collector:   collector,
		balance:     balance,
		transaction: transaction,
		activate:    activate,
	}
}

func (c *TransactionCollector) Collect(rec *observer.Record) *observer.Transaction {
	if rec == nil {
		return nil
	}
	res := c.collector.Collect(rec)
	act := c.activate.Collect(rec)
	var half *observer.Chain
	if isCreateTransaction(rec) {
		half = c.balance.Collect(rec)
	}

	if res != nil {
		half = c.balance.Collect(res)
	}

	var chain *observer.Chain
	if half != nil {
		chain = c.transaction.Collect(half)
	}

	if act != nil {
		chain = c.transaction.Collect(act)
	}

	if chain == nil {
		return nil
	}

	coupleAct := c.unwrapTransactionChain(chain)

	tx, err := c.build(coupleAct)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build transaction"))
		return nil
	}
	return tx
}

type Transaction struct {
	foundation.BaseContract
	Amount      uint64
	PulseTx     insolar.PulseNumber
	ExtTxId     string
	TxDirection TxDirection
	MemberRef   insolar.Reference
	GroupRef    insolar.Reference
	UID         string
	Status      StatusTx
}

func (c *TransactionCollector) build(act *observer.Activate) (*observer.Transaction, error) {
	if act == nil {
		return nil, errors.New("trying to create transaction from non complete builder")
	}

	var tx Transaction

	err := insolar.Deserialize(act.Virtual.GetActivate().Memory, &tx)
	if err != nil {
		return nil, err
	}

	date, err := act.ID.Pulse().AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert transaction create pulse (%d) to time", act.ID.Pulse())
	}

	fmt.Println("Collect new transaction ref:", act.ID.String())
	return &observer.Transaction{
		Reference:   *insolar.NewReference(act.ID),
		Amount:      strconv.FormatUint(tx.Amount, 10),
		Timestamp:   date.Unix(),
		ExtTxId:     tx.ExtTxId,
		TxDirection: tx.TxDirection.String(),
		GroupRef:    tx.GroupRef,
		MemberRef:   tx.MemberRef,
		UID:         tx.UID,
		Status:      tx.Status.String(),
	}, nil
}

func isTransactionCreationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}
	args := request.ParseMemberCallArguments()
	if args.Params.CallSite == "group.addTransaction" || args.Params.CallSite == "group.disburse" {
		return true
	}
	return false
}

func isTransactionNew(chain interface{}) bool {
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
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5gs8yv91EiGSWZK862DDoM7qJMXUnfjktXxYMYq") // transaction
	return in.Prototype.Equal(*prototypeRef)
}

func isTransactionActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5gs8yv91EiGSWZK862DDoM7qJMXUnfjktXxYMYq")
	return act.Image.Equal(*prototypeRef)
}

func isCreateTransaction(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "AddTransaction" && in.Method != "Disburse" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9") // group
	return in.Prototype.Equal(*prototypeRef)
}

func (c *TransactionCollector) unwrapTransactionChain(chain *observer.Chain) *observer.Activate {
	log := c.log

	coupledAct, ok := chain.Child.(*observer.CoupledActivate)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.CoupledActivate", reflect.TypeOf(chain.Child)))
		return nil
	}
	if coupledAct.Activate == nil {
		log.Error(errors.New("invalid coupled activate chain, child is nil"))
		return nil
	}

	return coupledAct.Activate
}

func transactionUpdate(act *observer.Record) (*Transaction, error) {
	var memory []byte
	switch v := act.Virtual.Union.(type) {
	case *record.Virtual_Activate:
		memory = v.Activate.Memory
	case *record.Virtual_Amend:
		memory = v.Amend.Memory
	default:
		log.Error(errors.New("invalid record to get transaction memory"))
	}

	if memory == nil {
		log.Warn(errors.New("transaction memory is nil"))
		return nil, errors.New("invalid record to get transaction memory")
	}

	var transaction Transaction

	err := insolar.Deserialize(memory, &transaction)
	if err != nil {
		log.Error(errors.New("failed to deserialize transaction memory"))
	}

	return &transaction, nil
}
