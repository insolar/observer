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

package collecting

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

const (
	GetFreeMigrationAddress = "GetFreeMigrationAddress"
)

type WastingCollector struct {
	fetcher store.RecordFetcher
}

func NewWastingCollector(fetcher store.RecordFetcher) *WastingCollector {
	return &WastingCollector{
		fetcher: fetcher,
	}
}

func (c *WastingCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Wasting {
	if rec == nil {
		return nil
	}

	result := observer.CastToResult(rec)
	if result == nil {
		return nil
	}

	if !result.IsSuccess() {
		return nil
	}

	requestID := result.Request()
	if requestID.IsEmpty() {
		panic(fmt.Sprintf("recordID %s: empty requestID from result", rec.ID.String()))
	}

	recordRequest, err := c.fetcher.Request(ctx, requestID)
	if err != nil {
		panic(errors.Wrapf(err, "recordID %s: failed to fetch request", rec.ID.String()))
	}

	request := recordRequest.Virtual.GetIncomingRequest()
	if request == nil {
		return nil
	}

	if request.Method != GetFreeMigrationAddress {
		return nil
	}

	address := ""
	result.ParseFirstPayloadValue(&address)
	return &observer.Wasting{
		Addr: address,
	}
}
