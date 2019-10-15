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

package grpc

import (
	"context"
	"io"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type RecordFetcher struct {
	log     *logrus.Logger
	client  exporter.RecordExporterClient
	records observer.RecordStorage
	request *exporter.GetRecords
}

func NewRecordFetcher(
	cfg *configuration.Configuration,
	obs *observability.Observability,
	client exporter.RecordExporterClient,
) *RecordFetcher {
	request := &exporter.GetRecords{Count: cfg.Replicator.BatchSize}
	return &RecordFetcher{
		log:     obs.Log(),
		client:  client,
		request: request,
	}
}

func (f *RecordFetcher) Fetch(pulse insolar.PulseNumber) ([]*observer.Record, insolar.PulseNumber, error) {
	ctx := context.Background()
	client := f.client

	f.request.PulseNumber = pulse
	f.request.RecordNumber = 0

	batch := []*observer.Record{}
	shouldIterateFrom := insolar.PulseNumber(0)
	for {
		stream, err := client.Export(ctx, f.request)
		if err != nil {
			return batch, shouldIterateFrom, errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method")
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return batch, 0, errors.Wrapf(err, "received error value from records gRPC stream %v", f.request)
			}

			if resp.ShouldIterateFrom != nil {
				shouldIterateFrom = *resp.ShouldIterateFrom
			}

			if resp.Record.ID.Pulse() != pulse {
				return batch, shouldIterateFrom, nil
			}
			model := (*observer.Record)(&resp.Record)
			batch = append(batch, model)

			f.request.PulseNumber = model.ID.Pulse()
			f.request.RecordNumber = resp.RecordNumber
		}
		if uint32(len(batch))%f.request.Count != 0 {
			return batch, shouldIterateFrom, nil
		}
	}
}
