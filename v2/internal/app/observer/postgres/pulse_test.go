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
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/internal/app/observer"
	"github.com/insolar/observer/v2/observability"
)

func TestPulseStorage_Insert(t *testing.T) {

	t.Run("nil", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		storage := NewPulseStorage(cfg, obs, nil)

		require.NoError(t, storage.Insert(nil))
	})

	t.Run("insert_with_err", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return nil, errors.New("something wrong")
		}
		storage := NewPulseStorage(cfg, obs, db)

		empty := &observer.Pulse{}

		require.Error(t, storage.Insert(empty))
	})

	t.Run("insert_with_conflict", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(obs.Log()), nil
		}
		storage := NewPulseStorage(cfg, obs, db)

		empty := &observer.Pulse{}

		require.NoError(t, storage.Insert(empty))
	})

	t.Run("empty", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		empty := &observer.Pulse{}
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.query = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(obs.Log(), empty), nil
		}
		storage := NewPulseStorage(cfg, obs, db)

		err := storage.Insert(empty)
		require.NoError(t, err)
	})
}

func TestPulseStorage_Last(t *testing.T) {
	t.Run("connection_error", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()

		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return nil, errors.New("dial tcp [::1]:5432: connect: connection refused")
		}
		cfg.DB.Attempts = 1
		cfg.DB.AttemptInterval = time.Nanosecond

		storage := NewPulseStorage(cfg, obs, db)
		require.Nil(t, storage.Last())
	})

	t.Run("no_pulses", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()

		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return nil, nil
		}
		cfg.DB.Attempts = 1
		cfg.DB.AttemptInterval = time.Nanosecond
		empty := &observer.Pulse{}

		storage := NewPulseStorage(cfg, obs, db)
		require.Equal(t, empty, storage.Last())
	})

	t.Run("connection_error_on_second_query", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		expected := &observer.Pulse{Number: insolar.GenesisPulse.PulseNumber}
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			switch reflect.TypeOf(model) {
			case reflect.TypeOf(orm.Scan()):
				m, err := orm.NewModel(model)
				require.NoError(t, err)
				buf := []byte(strconv.Itoa(1))
				err = m.ScanColumn(0, "count(*)", buf)
				require.NoError(t, err)
				return makeResult(obs.Log(), expected), nil
			}
			return nil, errors.New("dial tcp [::1]:5432: connect: connection refused")
		}
		cfg.DB.Attempts = 1
		cfg.DB.AttemptInterval = time.Nanosecond

		storage := NewPulseStorage(cfg, obs, db)
		require.Nil(t, storage.Last())
	})

	t.Run("existing_pulse", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		expected := &observer.Pulse{Number: insolar.GenesisPulse.PulseNumber}
		db := &dbMock{}
		db.model = func(model ...interface{}) *orm.Query {
			return orm.NewQuery(db, model...)
		}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			switch reflect.TypeOf(model) {
			case reflect.TypeOf(orm.Scan()):
				m, err := orm.NewModel(model)
				require.NoError(t, err)
				buf := []byte(strconv.Itoa(1))
				err = m.ScanColumn(0, "count(*)", buf)
				require.NoError(t, err)
				return makeResult(obs.Log(), expected), nil
			default:
				m, err := orm.NewModel(model)
				require.NoError(t, err)
				err = m.ScanColumn(0, "pulse", []byte(strconv.Itoa(int(expected.Number))))
				require.NoError(t, err)
				return makeResult(obs.Log(), expected), nil
			}
		}

		storage := NewPulseStorage(cfg, obs, db)
		require.Equal(t, expected, storage.Last())
	})
}
