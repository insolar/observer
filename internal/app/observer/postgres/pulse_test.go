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
// +build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/models"
	"github.com/insolar/observer/observability"
)

func TestPulseStorage_Insert(t *testing.T) {
	ctx := context.Background()
	t.Run("nil", func(t *testing.T) {
		obs := observability.Make(ctx)
		storage := NewPulseStorage(obs.Log(), nil)

		require.NoError(t, storage.Insert(nil))
	})

	t.Run("insert_with_err", func(t *testing.T) {
		obs := observability.Make(ctx)
		db := &DBMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return nil, errors.New("something wrong")
		}
		storage := NewPulseStorage(obs.Log(), db)

		empty := &observer.Pulse{}

		require.Error(t, storage.Insert(empty))
	})

	t.Run("insert_with_conflict", func(t *testing.T) {
		obs := observability.Make(ctx)
		db := &DBMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(obs.Log()), nil
		}
		storage := NewPulseStorage(obs.Log(), db)

		empty := &observer.Pulse{}

		require.NoError(t, storage.Insert(empty))
	})

	t.Run("empty", func(t *testing.T) {
		obs := observability.Make(ctx)
		empty := &observer.Pulse{}
		db := &DBMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(obs.Log(), empty), nil
		}
		storage := NewPulseStorage(obs.Log(), db)

		err := storage.Insert(empty)
		require.NoError(t, err)
	})
}

func TestPulseStorage_Last(t *testing.T) {
	ctx := context.Background()
	t.Run("connection_error", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make(ctx)

		db := &DBMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return nil, errors.New("dial tcp [::1]:5432: connect: connection refused")
		}
		cfg.DB.Attempts = 1
		cfg.DB.AttemptInterval = time.Nanosecond

		storage := NewPulseStorage(obs.Log(), db)
		_, err := storage.Last()
		require.Error(t, err)
	})

	t.Run("no_pulses", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make(ctx)

		db := &DBMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(obs.Log()), pg.ErrNoRows
		}
		cfg.DB.Attempts = 1
		cfg.DB.AttemptInterval = time.Nanosecond

		storage := NewPulseStorage(obs.Log(), db)
		pulse, err := storage.Last()
		require.Error(t, err)
		require.Nil(t, pulse)
	})

	t.Run("existing_pulse", func(t *testing.T) {
		obs := observability.Make(ctx)
		expected := &observer.Pulse{Number: insolar.GenesisPulse.PulseNumber}
		db := &DBMock{}
		db.model = func(model ...interface{}) *orm.Query {
			model[0].(*models.Pulse).Pulse = uint32(expected.Number)
			return orm.NewQuery(db, model...)
		}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			res := makeResult(obs.Log(), expected)
			return res, nil
		}

		storage := NewPulseStorage(obs.Log(), db)
		pulse, err := storage.Last()
		require.NoError(t, err)
		require.Equal(t, expected, pulse)
	})
}
