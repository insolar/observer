package grpc

import (
	"context"
	"io"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/cycle"
	"github.com/insolar/observer/observability"
)

//go:generate minimock -g -i github.com/insolar/insolar/ledger/heavy/exporter.RecordExporterClient -s "_mock.go"

type RecordFetcher struct {
	log     insolar.Logger
	cfg     *configuration.Observer
	client  exporter.RecordExporterClient
	records observer.RecordStorage //nolint: unused,structcheck
	request *exporter.GetRecords
}

func NewRecordFetcher(
	cfg *configuration.Observer,
	obs *observability.Observability,
	client exporter.RecordExporterClient,
) *RecordFetcher {
	request := &exporter.GetRecords{Count: cfg.Replicator.BatchSize}
	return &RecordFetcher{
		log:     obs.Log(),
		cfg:     cfg,
		client:  client,
		request: request,
	}
}

func (f *RecordFetcher) Fetch(
	ctx context.Context,
	pulse insolar.PulseNumber,
) (
	map[uint32]*exporter.Record,
	insolar.PulseNumber,
	error,
) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := f.client

	f.request.PulseNumber = pulse
	f.request.RecordNumber = 0

	batch := make(map[uint32]*exporter.Record)
	var (
		counter           uint32
		shouldIterateFrom = insolar.PulseNumber(0)
		attempts          = cycle.Limit(0)
	)
	// Get pulse batches
	for {
		counter = 0
		limitExceeded := false
		f.log.Debug("Data request: ", f.request)
		stream, err := client.Export(getCtxWithClientVersion(ctx), f.request)

		if err != nil {
			f.log.Debug("Data request failed: ", err)
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
				detectedDeprecatedVersion(err, f.log)
				f.log.Debugf("received error value from records gRPC stream %v", f.request)
				if connectivity.ExporterLimited(err) && attempts < f.cfg.Replicator.Attempts {
					limitExceeded = true
					f.log.WithFields(map[string]interface{}{
						"attempt":       attempts,
						"attempt_limit": f.cfg.Replicator.Attempts,
					}).Errorf("Exporter rate limit exceeded. Will try again in %s",
						f.cfg.Replicator.AttemptInterval.String())
					time.Sleep(f.cfg.Replicator.AttemptInterval)
					attempts++
					break
				}
				return batch, shouldIterateFrom, errors.Wrapf(err, "received error value from records gRPC stream %v", f.request)
			}

			// There is no records at all
			if resp.ShouldIterateFrom != nil {
				f.log.Debug("Received Should iterate from ", resp.ShouldIterateFrom.String())
				shouldIterateFrom = *resp.ShouldIterateFrom
				return batch, shouldIterateFrom, nil
			}

			// If we see next pulse, then stop iteration
			if resp.Record.ID.Pulse() != pulse {
				f.log.Debug("next pulse received ", resp.Record.ID.Pulse())
				// If we have no records in this pulse, then go to next not empty pulse
				if len(batch) == 0 {
					shouldIterateFrom = resp.Record.ID.Pulse()
					f.log.Debug("shouldIterateFrom set to ", shouldIterateFrom)
				}
				// todo we can read records by several pulses
				return batch, shouldIterateFrom, nil
			}
			batch[resp.RecordNumber] = resp

			counter++
			f.request.RecordNumber = resp.RecordNumber
		}

		f.log.Debug("go to next round, fetched: ", len(batch))
		if limitExceeded {
			continue
		}
		// If we get less than batch size, then stop
		if counter < f.request.Count {
			f.log.Debugf("Exiting: counter %+v", uint32(len(batch)))
			return batch, shouldIterateFrom, nil
		}
	}
}
