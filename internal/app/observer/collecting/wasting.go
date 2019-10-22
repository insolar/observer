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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

const (
	GetFreeMigrationAddress = "GetFreeMigrationAddress"
)

type WastingCollector struct {
	log     *logrus.Logger
	fetcher store.RecordFetcher
}

func NewWastingCollector(log *logrus.Logger, fetcher store.RecordFetcher) *WastingCollector {
	return &WastingCollector{
		log:     log,
		fetcher: fetcher,
	}
}

func (c *WastingCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Wasting {
	logger := c.log.WithField("collector", "wasting")
	if rec == nil {
		logger.Debug("empty record")
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

	recordRequest, err := c.fetcher.Request(ctx, requestID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to fetch request"))
		return nil
	}

	request := recordRequest.Virtual.GetIncomingRequest()
	if request == nil {
		logger.Debug("not a incoming request reason")
		return nil
	}

	if request.Method != GetFreeMigrationAddress {
		logger.Debug("not GetFreeMigrationAddress request")
		return nil
	}

	address := ""
	result.ParseFirstPayloadValue(&address)
	return &observer.Wasting{
		Addr: address,
	}
}
