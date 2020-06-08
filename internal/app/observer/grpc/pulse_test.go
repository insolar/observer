// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package grpc

import (
	"context"
	"io"
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

func TestPulseFetcher_Fetch(t *testing.T) {
	ctx := context.Background()
	t.Run("empty_stream", func(t *testing.T) {
		cfg := configuration.Observer{}.Default()
		obs := observability.Make(ctx)
		cfg.Replicator.AttemptInterval = 0
		cfg.Replicator.Attempts = 1
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return nil, io.EOF
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		_, err := fetcher.Fetch(ctx, 0)
		require.Equal(t, ErrNoPulseReceived, err)
	})

	t.Run("one_pulse", func(t *testing.T) {
		cfg := configuration.Observer{}.Default()
		obs := observability.Make(ctx)
		expected := &observer.Pulse{}
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return &exporter.Pulse{}, nil
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		pulse, err := fetcher.Fetch(ctx, 0)
		require.NoError(t, err)
		require.Equal(t, expected, pulse)
	})

	t.Run("ordinary", func(t *testing.T) {
		cfg := configuration.Observer{}.Default()
		obs := observability.Make(ctx)
		expected := &observer.Pulse{}
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return &exporter.Pulse{}, nil
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		pulse, err := fetcher.Fetch(ctx, 0)
		require.NoError(t, err)
		require.Equal(t, expected, pulse)
	})

	t.Run("failed_export", func(t *testing.T) {
		cfg := configuration.Observer{}.Default()
		obs := observability.Make(ctx)
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return nil, errors.New("failed export")
		}
		cfg.Replicator.AttemptInterval = 0
		cfg.Replicator.Attempts = 5
		fetcher := NewPulseFetcher(cfg, obs, client)

		require.Panics(t, func() {
			_, _ = fetcher.Fetch(ctx, 0)
		})
	})

	t.Run("failed_recv", func(t *testing.T) {
		cfg := configuration.Observer{}.Default()
		obs := observability.Make(ctx)
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return nil, errors.New("failed to get pulse")
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		require.Panics(t, func() {
			_, _ = fetcher.Fetch(ctx, 0)
		})
	})
}

func TestPulseFetcher_FetchCurrent(t *testing.T) {
	ctx := context.Background()
	t.Run("happy topsyncpulse", func(t *testing.T) {
		cfg := configuration.Observer{}.Default()
		obs := observability.Make(ctx)
		cfg.Replicator.AttemptInterval = 0
		cfg.Replicator.Attempts = 1
		pn := insolar.PulseNumber(10000)
		client := &pulseClient{}
		client.topSyncPulse = func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (response *exporter.TopSyncPulseResponse, e error) {
			return &exporter.TopSyncPulseResponse{
				Polymorph:   0,
				PulseNumber: pn.AsUint32(),
			}, nil
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		tsp, err := fetcher.FetchCurrent(ctx)
		require.NoError(t, err)
		require.Equal(t, pn, tsp)
	})

	t.Run("topsyncpulse returns error", func(t *testing.T) {
		cfg := configuration.Observer{}.Default()
		obs := observability.Make(ctx)
		cfg.Replicator.AttemptInterval = 0
		cfg.Replicator.Attempts = 1
		client := &pulseClient{}
		client.topSyncPulse = func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (response *exporter.TopSyncPulseResponse, e error) {
			return &exporter.TopSyncPulseResponse{}, errors.New("test")
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		require.Panics(t, func() {
			_, _ = fetcher.FetchCurrent(ctx)
		})
	})
}

type pulseClient struct {
	export       func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error)
	topSyncPulse func(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error)
}

func (c *pulseClient) Export(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
	return c.export(ctx, in, opts...)
}

func (c *pulseClient) TopSyncPulse(ctx context.Context, in *exporter.GetTopSyncPulse, opts ...grpc.CallOption) (*exporter.TopSyncPulseResponse, error) {
	return c.topSyncPulse(ctx, in, opts...)
}

type pulseStream struct {
	grpc.ClientStream
	recv func() (*exporter.Pulse, error)
}

func (s *pulseStream) Recv() (*exporter.Pulse, error) {
	return s.recv()
}
