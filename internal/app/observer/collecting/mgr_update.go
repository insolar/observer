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

type MGRUpdateCollector struct {
	log *logrus.Logger
}

func NewMGRUpdateCollector(log *logrus.Logger) *MGRUpdateCollector {
	return &MGRUpdateCollector{
		log: log,
	}
}

func (c *MGRUpdateCollector) Collect(rec *observer.Record) *observer.MGRUpdate {
	defer panic.Catch("group_update_collector")

	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isMGRAmend(v.Amend) {
		return nil
	}

	amd := rec.Virtual.GetAmend()
	mgr, err := mgrUpdate(rec)

	if err != nil {
		logrus.Info(err.Error())
		return nil
	}

	var seq []observer.Sequence
	for _, v := range mgr.Sequence {
		seq = append(seq, observer.Sequence{Member: v.Member, DueDate: v.DueDate, IsActive: v.IsActive})
	}

	return &observer.MGRUpdate{
		GroupReference:   mgr.GroupReference,
		PrevState:        *insolar.NewReference(amd.PrevState),
		MGRState:         *insolar.NewReference(rec.ID),
		StartRoundDate:   int64(mgr.StartRoundDate),
		FinishRoundDate:  int64(mgr.FinishRoundDate),
		AmountDue:        mgr.AmountDue,
		PaymentFrequency: mgr.PaymentFrequency,
		NextPaymentTime:  int64(mgr.NextPaymentTime),
		Sequence:         seq,
	}
}

func isMGRAmend(amd *record.Amend) bool {
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A6L4ytii4Z9jWLJpFqjDkH8ZRZ8HNscmmzsBF85i")
	return amd.Image.Equal(*prototypeRef)
}
