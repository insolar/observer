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
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestHeavy_Export(t *testing.T) {
	ctx := context.Background()
	limits := grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(1073741824),
		grpc.MaxCallSendMsgSize(1073741824),
	)
	conn, err := grpc.Dial("127.0.0.1:5679", limits, grpc.WithInsecure())
	require.NoError(t, err)

	pn, rn := insolar.PulseNumber(0), uint32(0)
	log.Infof("before cycle")
	for {
		client := exporter.NewRecordExporterClient(conn)
		req := &exporter.GetRecords{Count: 1000, PulseNumber: pn, RecordNumber: rn}
		log.Infof("before export call")
		stream, err := client.Export(ctx, req)
		// client := exporter.NewPulseExporterClient(conn)
		// req := &exporter.GetPulses{Count: 1, PulseNumber: 19627849}
		// stream, err := client.Export(ctx, req)
		if err != nil {
			log.Infof(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method").Error())
			return
		}
		for {
			fmt.Println("before recv")
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Error(errors.Wrapf(err, "received error value from gRPC stream"))
				break
			}
			if resp == nil {
				fmt.Println("resp is nil")
				return
			}
			// t.Logf("pulse %d", resp.PulseNumber)
			n, rec := resp.RecordNumber, &resp.Record
			pulse := rec.ID.Pulse()
			log.Infof("received %d %d %s \n", n, pulse, rec.ID.String())
			pn = pulse
			rn = n
		}
		log.Infof("stream finished")
		time.Sleep(1 * time.Second)
	}
}
