// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package grpc

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	insrecord "github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/observability"
)

func TestRecordFetcher_Fetch(t *testing.T) {
	mc := minimock.NewController(t)
	ctx := context.WithValue(context.Background(), configuration.VersionAPP, versionValue)
	cfg := configuration.Observer{}.Default()
	obs := observability.Make(ctx)
	recordClient := NewRecordExporterClientMock(mc)
	cfg.Replicator.AttemptInterval = 0
	cfg.Replicator.Attempts = 1

	t.Run("happy pulse", func(t *testing.T) {
		pn := insolar.PulseNumber(10000000)
		batchSize := 100
		totalRecs := 1111
		cfg.Replicator.BatchSize = uint32(batchSize)
		cnt := 0
		eoffed := true
		// This is like HME do
		generateRecords := func() (record *exporter.Record, e error) {
			if !eoffed && cnt%batchSize == 0 {
				eoffed = true
				return &exporter.Record{}, io.EOF
			}
			cnt++
			eoffed = false
			if cnt > totalRecs {
				return &exporter.Record{}, io.EOF
			}
			return &exporter.Record{
				RecordNumber: uint32(cnt),
				Record: insrecord.Material{
					ID: gen.IDWithPulse(pn),
				},
				ShouldIterateFrom: nil,
			}, nil
		}
		stream := recordStream{
			recv: generateRecords,
		}

		recordClient.funcExport = func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (r1 exporter.RecordExporter_ExportClient, err error) {
			require.Equal(t, pn, in.PulseNumber)
			require.Equal(t, cfg.Replicator.BatchSize, in.Count)
			return stream, nil
		}
		fetcher := NewRecordFetcher(cfg, obs, recordClient)
		recs, sif, err := fetcher.Fetch(ctx, pn)
		require.NoError(t, err)
		require.Len(t, recs, totalRecs)
		require.Equal(t, insolar.PulseNumber(0), sif)
	})

	t.Run("error after receive 50", func(t *testing.T) {
		pn := gen.PulseNumber()
		batchSize := 100
		totalRecs := 50
		cfg.Replicator.BatchSize = uint32(batchSize)
		cnt := 0
		generateRecords := func() (record *exporter.Record, e error) {
			cnt++
			if cnt > totalRecs {
				return &exporter.Record{}, errors.New("test")
			}
			return &exporter.Record{
				RecordNumber: uint32(cnt),
				Record: insrecord.Material{
					ID: gen.IDWithPulse(pn),
				},
				ShouldIterateFrom: nil,
			}, nil
		}
		stream := recordStream{
			recv: generateRecords,
		}

		recordClient.funcExport = func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (r1 exporter.RecordExporter_ExportClient, err error) {
			require.Equal(t, pn, in.PulseNumber)
			require.Equal(t, cfg.Replicator.BatchSize, in.Count)
			return stream, nil
		}
		fetcher := NewRecordFetcher(cfg, obs, recordClient)
		recs, sif, err := fetcher.Fetch(ctx, pn)
		require.Error(t, err)
		require.Len(t, recs, totalRecs)
		require.Equal(t, insolar.PulseNumber(0), sif)
	})

	t.Run("no records on heavy", func(t *testing.T) {
		pn := gen.PulseNumber()
		shouldIterFrom := gen.PulseNumber()
		batchSize := 100
		totalRecs := 0
		cfg.Replicator.BatchSize = uint32(batchSize)
		generateRecords := func() (record *exporter.Record, e error) {
			return &exporter.Record{
				ShouldIterateFrom: &shouldIterFrom,
			}, nil
		}
		stream := recordStream{
			recv: generateRecords,
		}

		recordClient.funcExport = func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (r1 exporter.RecordExporter_ExportClient, err error) {
			require.Equal(t, pn, in.PulseNumber)
			require.Equal(t, cfg.Replicator.BatchSize, in.Count)
			return stream, nil
		}
		fetcher := NewRecordFetcher(cfg, obs, recordClient)
		recs, sif, err := fetcher.Fetch(ctx, pn)
		require.NoError(t, err)
		require.Len(t, recs, totalRecs)
		require.Equal(t, shouldIterFrom, sif)
	})

	t.Run("happy with another pulse's records", func(t *testing.T) {
		pn := insolar.PulseNumber(10000000)
		batchSize := 100
		totalRecs := 100
		recsInThisPulse := 78
		cfg.Replicator.BatchSize = uint32(batchSize)
		cnt := 0
		eoffed := true
		generateRecords := func() (record *exporter.Record, e error) {
			pulseNumber := pn
			// HME can give records not for only requested pulse, if there are records in next pulse, and batch size is enough
			if cnt >= recsInThisPulse {
				pulseNumber = gen.PulseNumber()
			}
			if !eoffed && cnt%batchSize == 0 {
				eoffed = true
				return &exporter.Record{}, io.EOF
			}
			cnt++
			eoffed = false
			if cnt > totalRecs {
				return &exporter.Record{}, io.EOF
			}
			return &exporter.Record{
				RecordNumber: uint32(cnt),
				Record: insrecord.Material{
					ID: gen.IDWithPulse(pulseNumber),
				},
				ShouldIterateFrom: nil,
			}, nil
		}
		stream := recordStream{
			recv: generateRecords,
		}

		recordClient.funcExport = func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (r1 exporter.RecordExporter_ExportClient, err error) {
			require.Equal(t, pn, in.PulseNumber)
			require.Equal(t, cfg.Replicator.BatchSize, in.Count)
			return stream, nil
		}
		fetcher := NewRecordFetcher(cfg, obs, recordClient)
		recs, sif, err := fetcher.Fetch(ctx, pn)
		require.NoError(t, err)
		require.Len(t, recs, recsInThisPulse)
		require.Equal(t, insolar.PulseNumber(0), sif)
	})

	t.Run("fast forwarding on empty pulses", func(t *testing.T) {
		pn := insolar.PulseNumber(10000000)
		nextActivePN := insolar.PulseNumber(10010000)
		batchSize := 100
		recsInThisPulse := 0
		cfg.Replicator.BatchSize = uint32(batchSize)
		generateRecords := func() (record *exporter.Record, e error) {
			return &exporter.Record{
				RecordNumber: 0,
				Record: insrecord.Material{
					ID: gen.IDWithPulse(nextActivePN),
				},
				ShouldIterateFrom: nil,
			}, nil
		}
		stream := recordStream{
			recv: generateRecords,
		}

		recordClient.funcExport = func(ctx context.Context, in *exporter.GetRecords, opts ...grpc.CallOption) (r1 exporter.RecordExporter_ExportClient, err error) {
			require.Equal(t, pn, in.PulseNumber)
			require.Equal(t, cfg.Replicator.BatchSize, in.Count)
			return stream, nil
		}
		fetcher := NewRecordFetcher(cfg, obs, recordClient)
		recs, sif, err := fetcher.Fetch(ctx, pn)
		require.NoError(t, err)
		require.Len(t, recs, recsInThisPulse)
		require.Equal(t, nextActivePN, sif)
	})
}

type recordStream struct {
	grpc.ClientStream
	recv func() (*exporter.Record, error)
}

func (s recordStream) Recv() (*exporter.Record, error) {
	return s.recv()
}
