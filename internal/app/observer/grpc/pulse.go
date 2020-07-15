// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package grpc

import (
	"context"
	"io"

	"google.golang.org/grpc/metadata"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/cycle"
	"github.com/insolar/observer/observability"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
)

var (
	ErrNoPulseReceived = errors.New("No pulse received")
)

type PulseFetcher struct {
	cfg    *configuration.Observer
	log    insolar.Logger
	client exporter.PulseExporterClient
}

func NewPulseFetcher(
	cfg *configuration.Observer,
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
		resp *exporter.Pulse
	)
	version := ctx.Value(configuration.VersionAPP).(string)
	requestCtx, cancel := context.WithCancel(ctx)

	defer cancel()
	cycle.UntilError(func() error {
		stream, err := client.Export(metadata.AppendToOutgoingContext(requestCtx, configuration.VersionAPP.String(), version), request)
		if err != nil {
			f.log.WithField("request", request).
				Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method"))
			return err
		}

		resp, err = stream.Recv()
		if err != nil {
			// stream is closed, no point of retrying
			if err == io.EOF {
				f.log.Debug("EOF received, quit")
				return nil
			}
			f.log.WithField("request", request).
				Error(errors.Wrapf(err, "received error value from pulses gRPC stream"))
		}

		return err
	}, f.cfg.Replicator.AttemptInterval, f.cfg.Replicator.Attempts)

	if resp == nil {
		return nil, ErrNoPulseReceived
	}

	model := &observer.Pulse{
		Number:    resp.PulseNumber,
		Entropy:   resp.Entropy,
		Timestamp: resp.PulseTimestamp,
		Nodes:     resp.Nodes,
	}
	f.log.Debug("Received pulse ", model.Number)
	return model, nil
}

func (f *PulseFetcher) FetchCurrent(ctx context.Context) (insolar.PulseNumber, error) {
	client := f.client
	request := &exporter.GetTopSyncPulse{}
	tsp := &exporter.TopSyncPulseResponse{}
	var err error
	f.log.Debug("Fetching top sync pulse")
	version := ctx.Value(configuration.VersionAPP).(string)

	cycle.UntilError(func() error {
		tsp, err = client.TopSyncPulse(metadata.AppendToOutgoingContext(ctx, configuration.VersionAPP.String(), version), request)
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
