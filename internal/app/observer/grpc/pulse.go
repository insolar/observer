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

	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/cycle"
	"github.com/insolar/observer/observability"

	"github.com/insolar/observer/configuration"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

type PulseFetcher struct {
	cfg    *configuration.Configuration
	log    *logrus.Logger
	client exporter.PulseExporterClient
}

func NewPulseFetcher(
	cfg *configuration.Configuration,
	obs *observability.Observability,
	client exporter.PulseExporterClient) *PulseFetcher {
	return &PulseFetcher{
		cfg:    cfg,
		log:    obs.Log(),
		client: client,
	}
}

func (f *PulseFetcher) Fetch(ctx context.Context, last insolar.PulseNumber) (*observer.Pulse, error) {
	client := f.client
	request := &exporter.GetPulses{Count: 1, PulseNumber: last}
	f.log.Infof("Fetching %d pulses from %s", request.Count, last)
	var (
		err  error
		resp *exporter.Pulse
	)
	cycle.UntilError(func() error {
		var stream exporter.PulseExporter_ExportClient
		stream, err = client.Export(ctx, request)
		if err != nil {
			f.log.WithField("request", request).
				Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method"))
			return err
		}

		resp, err = stream.Recv()
		if err != nil {
			f.log.WithField("request", request).
				Error(errors.Wrapf(err, "received error value from pulses gRPC stream"))
		}

		return err
	}, f.cfg.Replicator.AttemptInterval, f.cfg.Replicator.Attempts)

	model := &observer.Pulse{
		Number:    resp.PulseNumber,
		Entropy:   resp.Entropy,
		Timestamp: resp.PulseTimestamp,
		Nodes:     resp.Nodes,
	}
	f.log.Debug("Received pulse ", model)
	return model, nil
}

func (f *PulseFetcher) FetchCurrent(ctx context.Context) (insolar.PulseNumber, error) {
	client := f.client
	request := &exporter.GetTopSyncPulse{}
	tsp := &exporter.TopSyncPulseResponse{}
	var err error
	f.log.Debug("Fetching top sync pulse")

	cycle.UntilError(func() error {
		tsp, err = client.TopSyncPulse(ctx, request)
		if err != nil {
			f.log.WithField("request", request).
				Error(errors.Wrapf(err, "failed to get tsp"))
			return err
		}
		return nil
	}, f.cfg.Replicator.AttemptInterval, f.cfg.Replicator.Attempts)

	f.log.Debug("Received top sync pulse ", tsp.PulseNumber)
	return insolar.PulseNumber(tsp.PulseNumber), nil
}
