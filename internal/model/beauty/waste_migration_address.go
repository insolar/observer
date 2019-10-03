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

package beauty

import (
	"context"

	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"

	"github.com/insolar/observer/internal/model"
)

type WasteMigrationAddress struct {
	Addr string
}

func (a *WasteMigrationAddress) Dump(ctx context.Context, tx orm.DB) error {
	res, err := tx.Model(&MigrationAddress{}).
		Where("addr=?", a.Addr).
		Set("wasted=true").
		Update()
	if err != nil {
		stats.Record(ctx, model.ErrorsCount.M(1))
		return errors.Wrapf(err, "failed to update migration address")
	}

	if res.RowsAffected() != 1 {
		stats.Record(ctx, model.ErrorsCount.M(1))
		return errors.Errorf("failed to update migration address rows_affected=%d", res.RowsAffected())
	}

	stats.Record(ctx, wastedMigrationAddresses.M(1))

	return nil
}
