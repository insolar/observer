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

package store

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

//go:generate minimock -i github.com/insolar/observer/internal/app/observer/store.RecordFetcher -o ./ -s _mock.go -g

type RecordFetcher interface {
	Request(ctx context.Context, reqID insolar.ID) (record.Material, error)
	Result(ctx context.Context, reqID insolar.ID) (record.Material, error)
	SideEffect(ctx context.Context, reqID insolar.ID) (record.Material, error)
	CalledRequests(ctx context.Context, reqID insolar.ID) ([]record.Material, error)
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer/store.RecordSetter -o ./ -s _mock.go -g

type RecordSetter interface {
	SetResult(ctx context.Context, record record.Material) error
	SetSideEffect(ctx context.Context, record record.Material) error
	SetRequest(ctx context.Context, record record.Material) error
	SetRequestBatch(ctx context.Context, requestRecords []record.Material) error
	SetResultBatch(ctx context.Context, requestRecords []record.Material) error
	SetSideEffectBatch(ctx context.Context, requestRecords []record.Material) error
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer/store.RecordStore -o ./ -s _mock.go -g

type RecordStore interface {
	RecordFetcher
	RecordSetter
}
