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
	"encoding/hex"

	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type RequestSchema struct {
	tableName struct{} `sql:"requests"` //nolint: unused,structcheck

	RequestID  string `sql:"request_id,pk"`
	Caller     string
	ReturnMode string
	Base       string
	Object     string
	Prototype  string
	Method     string
	Arguments  string
	Reason     string
}

type RequestStorage struct {
	log          insolar.Logger
	errorCounter prometheus.Counter
	db           orm.DB
}

func NewRequestStorage(obs *observability.Observability, db orm.DB) *RequestStorage {
	errorCounter := obs.Counter(prometheus.CounterOpts{
		Name: "observer_request_storage_error_counter",
		Help: "",
	})
	return &RequestStorage{
		log:          obs.Log(),
		errorCounter: errorCounter,
		db:           db,
	}
}

func (s *RequestStorage) Insert(model *observer.Request) error {
	if model == nil {
		s.log.Warnf("trying to insert nil request model")
		return nil
	}
	row := requestSchema(model)
	res, err := s.db.Model(row).
		OnConflict("DO NOTHING").
		Insert()

	if err != nil {
		return errors.Wrapf(err, "failed to insert request %v", row)
	}

	if res.RowsAffected() == 0 {
		s.errorCounter.Inc()
		s.log.WithField("request_row", row).
			Errorf("failed to insert request")
		return errors.New("failed to insert, affected is 0")
	}
	return nil
}

func requestSchema(model *observer.Request) *RequestSchema {
	req := model.Virtual.GetIncomingRequest()
	base, object, prototype := "", "", ""
	if nil != req.Base {
		base = req.Base.String()
	}
	if nil != req.Object {
		object = req.Object.String()
	}
	if nil != req.Prototype {
		prototype = req.Prototype.String()
	}
	return &RequestSchema{
		RequestID:  model.ID.String(),
		Caller:     req.Caller.String(),
		ReturnMode: req.ReturnMode.String(),
		Base:       base,
		Object:     object,
		Prototype:  prototype,
		Method:     req.Method,
		Arguments:  hex.EncodeToString(req.Arguments),
		Reason:     req.Reason.String(),
	}
}
