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

type KYCCollector struct {
	log *logrus.Logger
}

func NewKYCCollector(log *logrus.Logger) *KYCCollector {
	return &KYCCollector{
		log: log,
	}
}

func (c *KYCCollector) Collect(rec *observer.Record) *observer.UserKYC {
	defer panic.Catch("kyc_update_collector")

	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isUserAmend(v.Amend) {
		return nil
	}
	amd := rec.Virtual.GetAmend()
	kyc, time, source, err := userKYC(rec)

	if err != nil {
		return nil
	}

	return &observer.UserKYC{
		PrevState: amd.PrevState,
		UserState: rec.ID,
		KYC:       kyc,
		Timestamp: time,
		Source:    source,
	}
}

func isUserAmend(amd *record.Amend) bool {
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A5tDgkPiUrCANU8NTa73b7w6pWGRAUxJTYFXwTnR")
	return amd.Image.Equal(*prototypeRef)
}
