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

package pg

import (
	"context"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer/store"
)

type Store struct {
	db *pg.DB
}

func NewPgStore(db *pg.DB) *Store {
	return &Store{db: db}
}

type RawRequest struct {
	ID       string `sql:"request_id"`
	ReasonID string `sql:"reason_id"`
	Body     []byte `sql:"request_body"`
}

type RawResult struct {
	RequestID string `sql:"request_id"`
	Body      []byte `sql:"result_body"`
}

type RawSideEffect struct {
	RequestID string `sql:"request_id"`
	Body      []byte `sql:"side_effect_body"`
}

func (s *Store) Request(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	res := record.Material{}
	request := RawRequest{}
	_, err := s.db.QueryOneContext(ctx, &request, "select * from raw_requests where request_id = ?", reqID.String())
	if err != nil {
		if err == pg.ErrNoRows {
			return res, store.ErrNotFound
		}
		return res, errors.Wrap(err, "failed to fetch request")
	}

	err = res.Unmarshal(request.Body)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal request body")
	}

	return res, nil
}

func (s *Store) Result(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	res := record.Material{}
	result := RawResult{}
	_, err := s.db.QueryOneContext(ctx, &result, "select * from raw_results where request_id = ?", reqID.String())
	if err != nil {
		if err == pg.ErrNoRows {
			return res, store.ErrNotFound
		}
		return res, errors.Wrap(err, "failed to fetch result")
	}

	err = res.Unmarshal(result.Body)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal result body")
	}

	return res, nil
}

func (s *Store) SideEffect(ctx context.Context, reqID insolar.ID) (record.Material, error) {
	res := record.Material{}
	result := RawSideEffect{}
	_, err := s.db.QueryOneContext(ctx, &result, "select * from raw_side_effects where request_id = ?", reqID.String())
	if err != nil {
		if err == pg.ErrNoRows {
			return res, store.ErrNotFound
		}
		return res, errors.Wrap(err, "failed to fetch side effect")
	}

	err = res.Unmarshal(result.Body)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal side effect body")
	}

	return res, nil
}

func (s *Store) CalledRequests(ctx context.Context, reqID insolar.ID) ([]record.Material, error) {
	var res []record.Material

	result := make([]RawRequest, 0)
	dbResult, err := s.db.QueryContext(ctx, &result, "select * from raw_requests where reason_id = ?", reqID.String())
	if err != nil {
		return res, errors.Wrap(err, "failed to fetch side effect")
	}

	res = make([]record.Material, dbResult.RowsReturned())
	for i := range result {
		err = res[i].Unmarshal(result[i].Body)
		if err != nil {
			return res, errors.Wrap(err, "failed to unmarshal side effect body")
		}
	}

	return res, nil
}

func (s *Store) SetResult(ctx context.Context, resultRecord record.Material) error {
	if resultRecord.Virtual.GetResult() == nil {
		return errors.Errorf("trying to save not a result as result")
	}
	id, err := store.RequestID(&resultRecord)
	if err != nil {
		return errors.Wrap(err, "failed to parse result data")
	}

	body, err := resultRecord.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to marshal result")
	}

	_, err = s.db.ExecContext(ctx, `insert into raw_results (request_id, result_body) values (?, ?)
                                           ON CONFLICT DO NOTHING`,
		id.String(), body)

	return errors.Wrap(err, "failed to insert result")
}

func (s *Store) SetSideEffect(ctx context.Context, sideEffectRecord record.Material) error {
	if sideEffectRecord.Virtual.GetAmend() == nil &&
		sideEffectRecord.Virtual.GetActivate() == nil &&
		sideEffectRecord.Virtual.GetDeactivate() == nil {
		return errors.Errorf("trying to save not a side effect as side effect")
	}

	id, err := store.RequestID(&sideEffectRecord)
	if err != nil {
		return errors.Wrap(err, "failed to parse side effect data")
	}

	body, err := sideEffectRecord.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to marshal side effect")
	}

	_, err = s.db.ExecContext(ctx, `insert into raw_side_effects (request_id, side_effect_body) values (?, ?)
                                           ON CONFLICT DO NOTHING`,
		id.String(), body)

	return errors.Wrap(err, "failed to insert side effect")
}

func (s *Store) SetRequest(ctx context.Context, requestRecord record.Material) error {
	id, reason, err := store.ExtractRequestData(&requestRecord)
	if err != nil {
		return errors.Wrap(err, "failed to parse request data")
	}

	body, err := requestRecord.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	_, err = s.db.ExecContext(ctx, `insert into raw_requests (request_id, reason_id, request_body) values (?, ?, ?)
                                           ON CONFLICT DO NOTHING`,
		id.String(), reason.String(), body)

	return errors.Wrap(err, "failed to insert request")
}
