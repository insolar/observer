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

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"

	"github.com/insolar/observer/v2/internal/app/observer"
)

type PulseFetcher struct {
	client  exporter.PulseExporterClient
	pulses  observer.PulseStorage
	request *exporter.GetPulses
}

func NewPulseFetcher(client exporter.PulseExporterClient, pulses observer.PulseStorage) *PulseFetcher {
	last := pulses.Last()
	request := &exporter.GetPulses{Count: 1}
	if last != nil {
		pulse := last.Number
		request.PulseNumber = pulse
	}
	return &PulseFetcher{
		client:  client,
		request: request,
	}
}

func (f *PulseFetcher) Fetch() (*observer.Pulse, error) {
	ctx := context.Background()
	client := f.client
	stream, err := client.Export(ctx, f.request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method")
	}

	resp, err := stream.Recv()
	if err == io.EOF {
		return nil, errors.Wrapf(err, "HME returns empty pulse stream")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "received error value from pulses gRPC stream %v", f.request)
	}
	model := &observer.Pulse{
		Number:    resp.PulseNumber,
		Entropy:   resp.Entropy,
		Timestamp: resp.PulseTimestamp,
	}

	f.request.PulseNumber = model.Number
	return model, nil
}
