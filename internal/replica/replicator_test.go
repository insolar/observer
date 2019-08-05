package replica

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
