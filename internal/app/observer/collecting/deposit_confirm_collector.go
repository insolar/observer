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
	"context"

	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
)

type DepositMemberCollector struct {
	log insolar.Logger
}

func NewDepositMemberCollector(log insolar.Logger) *DepositMemberCollector {
	return &DepositMemberCollector{
		log: log,
	}
}

func (c *DepositMemberCollector) Collect(ctx context.Context, rec *observer.Record) *observer.DepositMemberUpdate {
	if rec == nil {
		return nil
	}

	log := c.log.WithField("recordID", rec.ID.String()).WithField("collector", "DepositMemberCollector")

	req := rec.Virtual.GetIncomingRequest()
	if req == nil {
		log.Debug("not an incoming request, skipping")
		return nil
	}
	if !isConfirmCall(req) {
		log.Debug("not a deposit confirm call, skipping")
		return nil
	}

	if req.Arguments == nil {
		log.Panic("empty arguments for confirm call")
	}

	var memberRef insolar.Reference
	err := insolar.Deserialize(req.Arguments, []interface{}{nil, nil, nil, nil, nil, &memberRef})
	if err != nil {
		panic(errors.Wrap(err, "couldn't parse arguments"))
	}

	log.Debugf("update %s: member %s", req.Object.String(), memberRef.String())

	return &observer.DepositMemberUpdate{
		Ref:    *req.Object,
		Member: memberRef,
	}
}

func isConfirmCall(req *record.IncomingRequest) bool {
	if req.Method != "Confirm" || req.CallType != record.CTMethod {
		return false
	}
	if !req.Prototype.Equal(*proxyDeposit.PrototypeReference) {
		return false
	}
	return true
}
