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
	"testing"

	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/internal/app/observer"
)

func Test_pulseSchema(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.Nil(t, pulseSchema(nil))
	})
}

func TestPulseStorage_Insert(t *testing.T) {
	db := &dbMock{}
	db.insert = func(model ...interface{}) error {
		return nil
	}

	t.Run("nil", func(t *testing.T) {
		log := logrus.New()

		storage := NewPulseStorage(log, db)

		require.NoError(t, storage.Insert(nil))
	})

	t.Run("empty", func(t *testing.T) {
		log := logrus.New()

		storage := NewPulseStorage(log, db)

		empty := &observer.Pulse{}

		err := storage.Insert(empty)
		require.NoError(t, err)
	})
}

func TestPulseStorage_Last(t *testing.T) {
	db := &dbMock{}
	db.model = func(model ...interface{}) *orm.Query {
		return orm.NewQuery(db, model...)
	}

	t.Run("no_pulses", func(t *testing.T) {
		log := logrus.New()
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return nil, errors.New("pg: no rows in result set")
		}

		storage := NewPulseStorage(log, db)
		require.Nil(t, storage.Last())
	})

	t.Run("existing_pulse", func(t *testing.T) {
		log := logrus.New()
		expected := &observer.Pulse{}
		db.queryOne = func(model, query interface{}, params ...interface{}) (orm.Result, error) {
			return makeResult(log, expected), nil
		}

		storage := NewPulseStorage(log, db)
		require.Equal(t, expected, storage.Last())
	})
}
