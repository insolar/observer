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

package replica

import (
	"context"
	"io"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/insolar/observer/internal/configuration"
	log "github.com/sirupsen/logrus"
)

type Handle func(rec *record.Material)

type Publisher interface {
	Subscribe(handle Handle)
}

type Replicator struct {
	cfg      *configuration.Configuration
	handlers []Handle

	conn *grpc.ClientConn
	stop chan bool
}

func NewReplicator() *Replicator {
	config := &configuration.Configuration{}
	cfg := config.Default()
	return &Replicator{cfg: cfg, stop: make(chan bool)}
}

func (r *Replicator) Subscribe(handle Handle) {
	r.handlers = append(r.handlers, handle)
}

func (r *Replicator) Init(ctx context.Context) error {
	limits := grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(r.cfg.Replicator.MaxTransportMsg),
		grpc.MaxCallSendMsgSize(r.cfg.Replicator.MaxTransportMsg),
	)
	conn, err := grpc.Dial(r.cfg.Replicator.Addr, limits, grpc.WithInsecure())
	if err != nil {
		log.Error(errors.New("failed to connect to HME"))
		return nil
	}
	r.conn = conn
	return nil
}

func (r *Replicator) Start(ctx context.Context) error {
	req := &exporter.GetRecords{Count: r.cfg.Replicator.BatchSize, PulseNumber: 0, RecordNumber: 0}
	go func() {
		for {
			last, lastPulse, count := r.getData(req)
			if count == 0 {
				time.Sleep(r.cfg.Replicator.RequestDelay)
			}
			req = &exporter.GetRecords{Count: r.cfg.Replicator.BatchSize, PulseNumber: lastPulse, RecordNumber: last}

			if r.needStop() {
				return
			}
		}
	}()
	return nil
}

func (r *Replicator) Stop(ctc context.Context) error {
	close(r.stop)
	if err := r.conn.Close(); err != nil {
		log.Error(errors.Wrapf(err, "failed to close grpc connection"))
	}
	return nil
}

func (r *Replicator) getData(req *exporter.GetRecords) (uint32, insolar.PulseNumber, int) {
	ctx := context.Background()
	count := 0
	last, lastPulse := req.RecordNumber, req.PulseNumber
	client := exporter.NewRecordExporterClient(r.conn)
	stream, err := client.Export(ctx, req)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to get gRPC stream from expoter.Export method"))
		return last, lastPulse, count
	}
	for {
		resp, err := stream.Recv()
		if resp != nil {
			log.Infof("repl: %v %v", resp.RecordNumber, resp.Record.ID.String())
		} else {
			log.Infof("repl: %v", err)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			// log.Error(errors.Wrapf(err, "failed to get value from gRPC stream"))
			break
		}
		n, rec := resp.RecordNumber, &resp.Record
		r.notify(rec)
		last, lastPulse = n, rec.ID.Pulse()
		count++
	}
	return last, lastPulse, count
}

func (r *Replicator) notify(rec *record.Material) {
	for _, handle := range r.handlers {
		handle(rec)
	}
}

func (r *Replicator) needStop() bool {
	select {
	case <-r.stop:
		return true
	default:
		// continue
	}
	return false
}
