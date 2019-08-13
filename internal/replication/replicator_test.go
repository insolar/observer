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

package replication

import (
	"context"
	"io"
	"testing"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func TestHeavy_Export(t *testing.T) {
	ctx := context.Background()
	limits := grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(1073741824),
		grpc.MaxCallSendMsgSize(1073741824),
	)
	conn, err := grpc.Dial("127.0.0.1:5678", limits, grpc.WithInsecure())
	client := exporter.NewRecordExporterClient(conn)
	req := &exporter.GetRecords{Count: 1000, PulseNumber: 0, RecordNumber: 0}
	stream, err := client.Export(ctx, req)
	if err != nil {
		t.Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method"))
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Error(errors.Wrapf(err, "received error value from gRPC stream"))
			break
		}
		n, rec := resp.RecordNumber, &resp.Record
		pulse := rec.ID.Pulse()
		t.Logf("received %d %d %s", n, pulse, rec.ID.String())
	}
	t.Logf("stream finished")
}
