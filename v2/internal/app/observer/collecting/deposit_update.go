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
	"github.com/insolar/insolar/logicrunner/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/insolar/logicrunner/builtin/proxy/deposit"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/internal/pkg/panic"
)

type DepositUpdateCollector struct {
	log *logrus.Logger
}

func NewDepositUpdateCollector(log *logrus.Logger) *DepositUpdateCollector {
	return &DepositUpdateCollector{
		log: log,
	}
}

func (c *DepositUpdateCollector) Collect(rec *observer.Record) *observer.DepositUpdate {
	defer panic.Catch("deposit_update_collector")

	if rec == nil {
		return nil
	}

	if !isDepositAmend(rec) {
		return nil
	}

	amd := rec.Virtual.GetAmend()
	d := c.depositState(amd)
	releaseTimestamp := int64(0)
	if holdReleadDate, err := d.PulseDepositUnHold.AsApproximateTime(); err == nil {
		releaseTimestamp = holdReleadDate.Unix()
	}

	return &observer.DepositUpdate{
		ID:              rec.ID,
		HoldReleaseDate: releaseTimestamp,
		Amount:          d.Amount,
		Balance:         d.Balance,
		PrevState:       amd.PrevState,
	}
}

func isDepositAmend(rec *observer.Record) bool {
	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return false
	}

	return v.Amend.Image.Equal(*proxyDeposit.PrototypeReference)
}

func (c *DepositUpdateCollector) depositState(amd *record.Amend) *deposit.Deposit {
	log := c.log
	d := deposit.Deposit{}
	err := insolar.Deserialize(amd.Memory, &d)
	if err != nil {
		log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}
