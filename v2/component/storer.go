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

package component

import (
	"github.com/go-pg/pg"

	"github.com/insolar/observer/v2/connectivity"
	"github.com/insolar/observer/v2/internal/app/observer/postgres"
	"github.com/insolar/observer/v2/observability"
)

func makeStorer(obs *observability.Observability, conn *connectivity.Connectivity) func(*beauty) {
	log := obs.Log()
	db := conn.PG()

	metric := observability.MakeBeautyMetrics(obs, "stored")
	return func(b *beauty) {
		if b == nil {
			return
		}

		err := db.RunInTransaction(func(tx *pg.Tx) error {
			transfers := postgres.NewTransferStorage(obs, tx)
			for _, transfer := range b.transfers {
				err := transfers.Insert(transfer)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Error(err)
			return
		}

		metric.Transfers.Add(float64(len(b.transfers)))
	}
}
