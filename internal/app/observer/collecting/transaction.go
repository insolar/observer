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
	"strconv"
)

type TransactionCollector struct {
	log *logrus.Logger
}

func NewTransactionCollector(log *logrus.Logger) *TransactionCollector {
	return &TransactionCollector{
		log: log,
	}
}

func (c *TransactionCollector) Collect(rec *observer.Record) *observer.Transaction {
	if rec == nil {
		return nil
	}
	actCandidate := observer.CastToActivate(rec)

	if !actCandidate.IsActivate() {
		return nil
	}

	act := actCandidate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5gs8yv91EiGSWZK862DDoM7qJMXUnfjktXxYMYq")
	if !act.Image.Equal(*prototypeRef) {
		return nil
	}

	tx, err := c.build(actCandidate)
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
