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
	records observer.RecordStorage //nolint: unused,structcheck
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

	var batch []*observer.Record
	var counter uint32
	shouldIterateFrom := insolar.PulseNumber(0)
	// Get pulse batches
	for {
		counter = 0
		f.log.Debug("Data request: ", f.request)
		stream, err := client.Export(ctx, f.request)

		if err != nil {
			return batch, shouldIterateFrom, errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method")
		}

		// Get batch
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				f.log.Debug("EOF received, quit")
				break
			}
			if err != nil {
				f.log.Debugf("received error value from records gRPC stream %v", f.request)
				return batch, shouldIterateFrom, errors.Wrapf(err, "received error value from records gRPC stream %v", f.request)
			}

			// There is no records at all
			if resp.ShouldIterateFrom != nil {
				f.log.Debug("Received Should iterate from ", resp.ShouldIterateFrom.String())
				err := stream.CloseSend()
				if err != nil {
					return batch, shouldIterateFrom, errors.Wrap(err, "error while closing gRPC stream")
				}
				shouldIterateFrom = *resp.ShouldIterateFrom
				return batch, shouldIterateFrom, nil
			}

			// If we see next pulse, then stop iteration
			if resp.Record.ID.Pulse() != pulse {
				f.log.Debug("wrong pulse received ", resp.Record.ID.Pulse())
				err := stream.CloseSend()
				if err != nil {
					return batch, shouldIterateFrom, errors.Wrap(err, "error while closing gRPC stream")
				}

				// If we have no records in this pulse, then go to next not empty pulse
				if len(batch) == 0 {
					shouldIterateFrom = resp.Record.ID.Pulse()
					f.log.Debug("shouldIterateFrom set to ", shouldIterateFrom)
				}
				return batch, shouldIterateFrom, nil
			}
			model := (*observer.Record)(&resp.Record)
			batch = append(batch, model)

			counter++
			f.request.RecordNumber = resp.RecordNumber
		}

		f.log.Debug("go to next round, fetched: ", len(batch))
		// If we get less than batch size, then stop
		if counter < f.request.Count {
			f.log.Debugf("Exiting: counter %+v", uint32(len(batch)))
			return batch, shouldIterateFrom, nil
		}
	}
}
