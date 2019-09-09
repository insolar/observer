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

	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/internal/app/observer"
)

type RecordFetcher struct {
	client  exporter.RecordExporterClient
	records observer.RecordStorage
	request *exporter.GetRecords
}

func NewRecordFetcher(
	cfg *configuration.Configuration,
	client exporter.RecordExporterClient,
	records observer.RecordStorage,
) *RecordFetcher {
	last := records.Last()
	request := &exporter.GetRecords{Count: cfg.Replicator.BatchSize}
	if last != nil {
		pulse := last.ID.Pulse()
		request.PulseNumber = pulse
		request.RecordNumber = records.Count(pulse)
	}
	return &RecordFetcher{
		client:  client,
		request: request,
	}
}

func (f *RecordFetcher) Fetch(pulse insolar.PulseNumber) ([]*observer.Record, error) {
	ctx := context.Background()
	client := f.client

	f.request.PulseNumber = pulse
	f.request.RecordNumber = 0
	stream, err := client.Export(ctx, f.request)
	if err != nil {
		return []*observer.Record{}, errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method")
	}

	batch := []*observer.Record{}
	for {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return batch, errors.Wrapf(err, "received error value from records gRPC stream %v", f.request)
			}
			if resp.Record.ID.Pulse() != pulse {
				return batch, nil
			}
			model := (*observer.Record)(&resp.Record)
			batch = append(batch, model)

			f.request.PulseNumber = model.ID.Pulse()
			f.request.RecordNumber = resp.RecordNumber
		}
		if f.request.Count != uint32(len(batch)) {
			return batch, nil
		}
	}
}
