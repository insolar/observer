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
	"strconv"
)

type SavingsUpdateCollector struct {
	log *logrus.Logger
}

func NewSavingsUpdateCollector(log *logrus.Logger) *SavingsUpdateCollector {
	return &SavingsUpdateCollector{
		log: log,
	}
}

func (c *SavingsUpdateCollector) Collect(rec *observer.Record) *observer.SavingUpdate {
	defer panic.Catch("savings_update_collector")

	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isSavingsAmend(v.Amend) {
		return nil
	}

	amd := rec.Virtual.GetAmend()
	saving, err := savingUpdate(rec)

	if err != nil {
		logrus.Info(err.Error())
		return nil
	}

	resMap := make(map[insolar.Reference]int64)

	for i, v := range saving.NSContribute {
		ref, err := insolar.NewReferenceFromString(i)
		if err != nil {
			logrus.Error(err)
			return nil
		}
		resMap[*ref], err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			logrus.Error(err)
			return nil
		}
	}

	resultProduct := observer.SavingUpdate{
		Reference:       *insolar.NewReference(rec.ObjectID),
		PrevState:       amd.PrevState,
		SavingState:     rec.ID,
		StartRoundDate:  saving.StartRoundDate,
		NextPaymentDate: saving.NextPaymentDate,
		NSContribute:    resMap,
	}

	return &resultProduct
}

func isSavingsAmend(amd *record.Amend) bool {
	prototypeRef, _ := insolar.NewReferenceFromString("0111A6Uo4DN71b7FVUjW6yZvPTm4JVk1rYdBpajUUig2")
	return amd.Image.Equal(*prototypeRef)
}
