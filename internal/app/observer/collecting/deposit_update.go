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

	"github.com/insolar/insolar/application/builtin/contract/deposit"
	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
)

type DepositUpdateCollector struct {
	log insolar.Logger
}

func NewDepositUpdateCollector(log insolar.Logger) *DepositUpdateCollector {
	return &DepositUpdateCollector{
		log: log,
	}
}

func (c *DepositUpdateCollector) Collect(ctx context.Context, rec *observer.Record) *observer.DepositUpdate {
	if rec == nil {
		return nil
	}

	log := c.log.WithField("recordID", rec.ID.String()).WithField("collector", "DepositUpdateCollector")

	if !isDepositAmend(rec) {
		return nil
	}

	amd := rec.Virtual.GetAmend()

	d := c.depositState(amd)

	log.Debugf("%s: amount %s, balance %s, txHash %s, prevState %s", rec.ID.String(), d.Amount, d.Balance, d.TxHash, amd.PrevState.String())

	res := &observer.DepositUpdate{
		ID:          rec.ID,
		Amount:      d.Amount,
		Balance:     d.Balance,
		PrevState:   amd.PrevState,
		TxHash:      d.TxHash,
		IsConfirmed: d.IsConfirmed,
	}

	if d.PulseDepositUnHold > 0 {
		holdReleasedDate, err := d.PulseDepositUnHold.AsApproximateTime()
		if err != nil {
			log.Error(errors.Wrap(err, "bad PulseDepositUnHold"))
		} else {
			res.HoldReleaseDate = holdReleasedDate.Unix()
			res.Timestamp = holdReleasedDate.Unix() - d.Lockup
		}
	}

	return res
}

func isDepositAmend(rec *observer.Record) bool {
	amd := rec.Virtual.GetAmend()
	if amd == nil {
		return false
	}

	return amd.Image.Equal(*proxyDeposit.PrototypeReference)
}

func (c *DepositUpdateCollector) depositState(amd *record.Amend) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(amd.Memory, &d)
	if err != nil {
		panic("failed to deserialize deposit contract state")
	}
	return &d
}
