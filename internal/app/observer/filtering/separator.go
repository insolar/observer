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

package filtering

import (
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/app/observer"
)

type SeparatorFilter struct{}

func NewSeparatorFilter() *SeparatorFilter {
	return &SeparatorFilter{}
}

func (*SeparatorFilter) Filter(
	records []*observer.Record,
) (
	requests []*observer.Request,
	results []*observer.Result,
	activates []*observer.Activate,
	amends []*observer.Amend,
	deactivates []*observer.Deactivate,
) {
	for _, rec := range records {
		switch rec.Virtual.Union.(type) {
		case *record.Virtual_IncomingRequest:
			req := observer.CastToRequest(rec)
			requests = append(requests, req)
		case *record.Virtual_Result:
			res := observer.CastToResult(rec)
			results = append(results, res)
		case *record.Virtual_Activate:
			act := observer.CastToActivate(rec)
			activates = append(activates, act)
		case *record.Virtual_Amend:
			amd := observer.CastToAmend(rec)
			amends = append(amends, amd)
		case *record.Virtual_Deactivate:
			deact := observer.CastToDeactivate(rec)
			deactivates = append(deactivates, deact)
		}
	}
	return
}
