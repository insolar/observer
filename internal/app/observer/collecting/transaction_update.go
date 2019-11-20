//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/panic"
	"github.com/sirupsen/logrus"
)

type TransactionUpdateCollector struct {
	log *logrus.Logger
}

func NewTransactionUpdateCollector(log *logrus.Logger) *TransactionUpdateCollector {
	return &TransactionUpdateCollector{
		log: log,
	}
}

func (c *TransactionUpdateCollector) Collect(rec *observer.Record) *observer.TransactionUpdate {
	defer panic.Catch("transaction_update_collector")

	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isTransactionAmend(v.Amend) {
		return nil
	}

	transaction, err := transactionUpdate(rec)

	if err != nil {
		logrus.Info(err.Error())
		return nil
	}

	date, err := rec.ID.GetPulseNumber().AsApproximateTime()
	if err != nil {
		logrus.Info(err.Error())
		return nil
	}

	return &observer.TransactionUpdate{
		Reference:   *insolar.NewReference(rec.ObjectID),
		Amount:      transaction.Amount,
		Timestamp:   date.Unix(),
		ExtTxId:     transaction.ExtTxId,
		TxDirection: transaction.TxDirection.String(),
		MemberRef:   transaction.MemberRef,
		GroupRef:    transaction.GroupRef,
		UID:         transaction.UID,
		Status:      transaction.Status.String(),
	}
}

func isTransactionAmend(amd *record.Amend) bool {
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5gs8yv91EiGSWZK862DDoM7qJMXUnfjktXxYMYq")
	return amd.Image.Equal(*prototypeRef)
}
