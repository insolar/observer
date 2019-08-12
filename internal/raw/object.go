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

package raw

import (
	"encoding/hex"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"

	"github.com/insolar/observer/internal/model/raw"
)

func parseActivate(rec *record.Material) *raw.Object {
	id := rec.ID
	act := rec.Virtual.GetActivate()
	return &raw.Object{
		ObjectID: insolar.NewReference(id).String(),
		Domain:   act.Domain.String(),
		Request:  act.Request.String(),
		Memory:   hex.EncodeToString(act.Memory),
		Image:    act.Image.String(),
		Parent:   act.Parent.String(),
		Type:     "ACTIVATE",
	}
}

func parseAmend(rec *record.Material) *raw.Object {
	id := rec.ID
	amend := rec.Virtual.GetAmend()
	return &raw.Object{
		ObjectID:  insolar.NewReference(id).String(),
		Domain:    amend.Domain.String(),
		Request:   amend.Request.String(),
		Memory:    hex.EncodeToString(amend.Memory),
		Image:     amend.Image.String(),
		PrevState: amend.PrevState.String(),
		Type:      "AMEND",
	}
}

func parseDeactivate(rec *record.Material) *raw.Object {
	id := rec.ID
	deact := rec.Virtual.GetDeactivate()
	return &raw.Object{
		ObjectID:  insolar.NewReference(id).String(),
		Domain:    deact.Domain.String(),
		Request:   deact.Request.String(),
		PrevState: deact.PrevState.String(),
		Type:      "DEACTIVATE",
	}
}
