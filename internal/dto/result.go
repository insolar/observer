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

package dto

import (
	"encoding/hex"
	"runtime/debug"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/model/raw"
)

type Result record.Material

func (r *Result) MapModel() *raw.Result {
	if r == nil {
		log.Errorf("trying to use nil dto.Result receiver")
		debug.PrintStack()
		return nil
	}
	res := r.Virtual.GetResult()
	return &raw.Result{
		ResultID: insolar.NewReference(r.ID).String(),
		Request:  res.Request.String(),
		Payload:  hex.EncodeToString(res.Payload),
	}
}
