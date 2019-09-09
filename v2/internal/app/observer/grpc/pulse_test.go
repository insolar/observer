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

	"github.com/insolar/observer/v2/internal/app/observer"
)

func TestPulseFetcher_Fetch(t *testing.T) {
	t.Run("empty_history", func(t *testing.T) {
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return nil, io.EOF
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		storage := &pulseStorage{}
		storage.last = func() *observer.Pulse {
			return nil
		}
		fetcher := NewPulseFetcher(client, storage)

		pulse, err := fetcher.Fetch()
		require.Error(t, err)
		require.Nil(t, pulse)
	})

	t.Run("empty_stream", func(t *testing.T) {
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return nil, io.EOF
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		storage := &pulseStorage{}
		storage.last = func() *observer.Pulse {
			return &observer.Pulse{}
		}
		fetcher := NewPulseFetcher(client, storage)

		pulse, err := fetcher.Fetch()
		require.Error(t, err)
		require.Nil(t, pulse)
	})

	t.Run("empty_history_and_one_pulse", func(t *testing.T) {
		expected := &observer.Pulse{}
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return &exporter.Pulse{}, nil
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		storage := &pulseStorage{}
		storage.last = func() *observer.Pulse {
			return nil
		}
		fetcher := NewPulseFetcher(client, storage)

		pulse, err := fetcher.Fetch()
		require.NoError(t, err)
		require.Equal(t, expected, pulse)
	})

	t.Run("ordinary", func(t *testing.T) {
		expected := &observer.Pulse{}
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return &exporter.Pulse{}, nil
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		storage := &pulseStorage{}
		storage.last = func() *observer.Pulse {
			return &observer.Pulse{}
		}
		fetcher := NewPulseFetcher(client, storage)

		pulse, err := fetcher.Fetch()
		require.NoError(t, err)
		require.Equal(t, expected, pulse)
	})

	t.Run("failed_export", func(t *testing.T) {
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return nil, errors.New("failed export")
		}
		storage := &pulseStorage{}
		storage.last = func() *observer.Pulse {
			return nil
		}
		fetcher := NewPulseFetcher(client, storage)

		pulse, err := fetcher.Fetch()
		require.Error(t, err)
		require.Nil(t, pulse)
	})

	t.Run("failed_recv", func(t *testing.T) {
		stream := &pulseStream{}
		stream.recv = func() (*exporter.Pulse, error) {
			return nil, errors.New("failed recv")
		}
		client := &pulseClient{}
		client.export = func(ctx context.Context, in *exporter.GetPulses, opts ...grpc.CallOption) (exporter.PulseExporter_ExportClient, error) {
			return stream, nil
		}
		storage := &pulseStorage{}
		storage.last = func() *observer.Pulse {
			return nil
		}
		fetcher := NewPulseFetcher(client, storage)

		pulse, err := fetcher.Fetch()
		require.Error(t, err)
		require.Nil(t, pulse)
	})
}

type pulseStorage struct {
	observer.PulseStorage
	last func() *observer.Pulse
}

func (s *pulseStorage) Last() *observer.Pulse {
	return s.last()
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
