// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

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
