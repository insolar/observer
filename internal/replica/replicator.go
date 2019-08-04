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

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/raw"
)

type Handle func(n uint32, rec *record.Material)

type Publisher interface {
	Subscribe(handle Handle)
}

type Replicator struct {
	Configurator configuration.Configurator `inject:""`
	cfg          *configuration.Configuration
	handlers     []Handle

	conn *grpc.ClientConn
	stop chan bool
}

func NewReplicator() *Replicator {
	return &Replicator{stop: make(chan bool)}
}

func (r *Replicator) Subscribe(handle Handle) {
	r.handlers = append(r.handlers, handle)
}

func (r *Replicator) Init(ctx context.Context) error {
	if r.Configurator != nil {
		r.cfg = r.Configurator.Actual()
	} else {
		r.cfg = configuration.Default()
	}
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
	last, lastPulse := r.getLastRecord()
	log.Infof("Starting replication from position (n: %d, pulse: %d)", last, lastPulse)
	req := &exporter.GetRecords{Count: r.cfg.Replicator.BatchSize, PulseNumber: lastPulse, RecordNumber: last}
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
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Debug(errors.Wrapf(err, "received error value from gRPC stream"))
			break
		}
		n, rec := resp.RecordNumber, &resp.Record
		r.notify(n, rec)
		last, lastPulse = n, rec.ID.Pulse()
		count++
	}
	return last, lastPulse, count
}

func (r *Replicator) notify(n uint32, rec *record.Material) {
	for _, handle := range r.handlers {
		handle(n, rec)
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

func (r *Replicator) getLastRecord() (uint32, insolar.PulseNumber) {
	opt, err := pg.ParseURL(r.cfg.DB.URL)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to parse cfg.DB.URL"))
		return 0, 0
	}
	db := pg.Connect(opt)
	rec := &raw.Record{}
	err = db.Model(rec).Last()
	if err != nil {
		log.Debug(errors.Wrapf(err, "failed to get last record from db"))
		return 0, 0
	}
	id := insolar.NewIDFromBytes(rec.Key)
	return rec.Number, id.Pulse()
}
