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

package postgres

import (
	"testing"

	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/observability"
)

func TestRecordStorage_Insert(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		obs := observability.Make()
		storage := NewRecordStorage(obs, nil)

		require.NoError(t, storage.Insert(nil))
	})

	t.Run("insert_with_err", func(t *testing.T) {
		obs := observability.Make()
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return nil, errors.New("something wrong")
		}
		storage := NewRecordStorage(obs, db)
		empty := &observer.Record{}

		require.Error(t, storage.Insert(empty))
	})

	t.Run("insert_with_conflict", func(t *testing.T) {
		obs := observability.Make()
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(obs.Log()), nil
		}
		storage := NewRecordStorage(obs, db)
		empty := &observer.Record{}

		require.NoError(t, storage.Insert(empty))
	})

	t.Run("empty", func(t *testing.T) {
		obs := observability.Make()
		empty := &observer.Record{}
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(obs.Log(), empty), nil
		}
		storage := NewRecordStorage(obs, db)

		err := storage.Insert(empty)
		require.NoError(t, err)
	})
}
