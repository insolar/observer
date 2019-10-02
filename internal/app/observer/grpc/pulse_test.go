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
	"testing"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

func TestPulseFetcher_Fetch(t *testing.T) {
	t.Run("empty_stream", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
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

		pulse, err := fetcher.Fetch(0)
		require.Error(t, err)
		require.Nil(t, pulse)
	})

	t.Run("one_pulse", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
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

		pulse, err := fetcher.Fetch(0)
		require.NoError(t, err)
		require.Equal(t, expected, pulse)
	})

	t.Run("ordinary", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
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

		pulse, err := fetcher.Fetch(0)
		require.NoError(t, err)
		require.Equal(t, expected, pulse)
	})

	t.Run("failed_export", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return nil, errors.New("failed export")
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		pulse, err := fetcher.Fetch(0)
		require.Error(t, err)
		require.Nil(t, pulse)
	})

	t.Run("failed_recv", func(t *testing.T) {
		cfg := configuration.Default()
		obs := observability.Make()
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return nil, errors.New("failed recv")
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		cfg.Replicator.Attempts = 1
		fetcher := NewPulseFetcher(cfg, obs, client)

		pulse, err := fetcher.Fetch(0)
		require.Error(t, err)
		require.Nil(t, pulse)
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
	pulse []*exporter.Pulse
	recv  func() (*exporter.Pulse, error)
}

func (s *pulseStream) Recv() (*exporter.Pulse, error) {
	return s.recv()
}
