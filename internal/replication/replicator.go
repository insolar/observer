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
	"sync"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/db"
	"github.com/insolar/observer/internal/model/raw"
)

type DataHandle func(uint32, *record.Material)

type OnData interface {
	SubscribeOnData(handle DataHandle)
}

type DumpHandle func(tx *pg.Tx, pub OnDumpSuccess) error

type OnDump interface {
	SubscribeOnDump(DumpHandle)
}

type SuccessHandle func()

type OnDumpSuccess interface {
	Subscribe(SuccessHandle)
}

type Replicator struct {
	Configurator     configuration.Configurator `inject:""`
	ConnectionHolder db.ConnectionHolder        `inject:""`
	cfg              *configuration.Configuration

	sync.RWMutex
	dataHandles []DataHandle
	dumpHandles []DumpHandle

	conn *grpc.ClientConn
	stop chan bool
}

func NewReplicator() *Replicator {
	return &Replicator{stop: make(chan bool)}
}

func (r *Replicator) SubscribeOnData(handle DataHandle) {
	r.Lock()
	defer r.Unlock()

	r.dataHandles = append(r.dataHandles, handle)
}

func (r *Replicator) SubscribeOnDump(handle DumpHandle) {
	r.Lock()
	defer r.Unlock()

	r.dumpHandles = append(r.dumpHandles, handle)
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
	current := r.currentPosition()
	log.Infof("Starting replication from position (rn: %d, pulse: %d)", current.rn, current.pn)
	req := r.makeRequest(current)
	go func() {
		for {
			batch, pos := r.pull(req)
			r.emit(current, batch)
			if len(batch) < int(r.cfg.Replicator.BatchSize) {
				if pos != current {
					r.emitDump(pos.pn)
				}
				time.Sleep(r.cfg.Replicator.RequestDelay)
			}
			current = pos
			req = r.makeRequest(current)

			if r.needStop() {
				return
			}
		}
	}()
	return nil
}

func (r *Replicator) pull(req *exporter.GetRecords) ([]dataMsg, position) {
	ctx := context.Background()
	pulse, number := req.PulseNumber, req.RecordNumber
	client := exporter.NewRecordExporterClient(r.conn)
	stream, err := client.Export(ctx, req)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to get gRPC stream from expoter.Export method"))
		return []dataMsg{}, position{pn: pulse, rn: number}
	}

	batch := []dataMsg{}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "received error value from gRPC stream"))
			break
		}
		rec, rn := &resp.Record, resp.RecordNumber
		batch = append(batch, dataMsg{rec: rec, rn: rn})
		pulse, number = rec.ID.Pulse(), rn
	}
	return batch, position{pn: pulse, rn: number}
}

func (r *Replicator) emit(last position, batch []dataMsg) {
	lastPulse := last.pn
	for _, msg := range batch {
		pn := msg.rec.ID.Pulse()
		if pn != lastPulse {
			r.emitDump(lastPulse)
			lastPulse = pn
		}
		r.emitData(msg.rn, msg.rec)
	}
}

func (r *Replicator) emitData(rn uint32, rec *record.Material) {
	r.RLock()
	defer r.RUnlock()

	for _, handle := range r.dataHandles {
		handle(rn, rec)
	}
}

func (r *Replicator) emitDump(pn insolar.PulseNumber) {
	db := r.ConnectionHolder.DB()
	emitter := &successEmitter{}
	log.Infof("pulse=%d", pn)
	for {
		err := db.RunInTransaction(func(tx *pg.Tx) error {
			for _, handle := range r.dumpHandles {
				if err := handle(tx, emitter); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to dump data"))
			time.Sleep(r.cfg.Replicator.TransactionRetryDelay)
			continue
		}
		break
	}
	// emitting success transaction
	emitter.emit()
}

type successEmitter struct {
	sync.Mutex
	handlers []func()
}

func (e *successEmitter) Subscribe(handle SuccessHandle) {
	e.Lock()
	defer e.Unlock()

	e.handlers = append(e.handlers, handle)
}

func (e *successEmitter) emit() {
	for _, h := range e.handlers {
		h()
	}
}

func (r *Replicator) Stop(ctc context.Context) error {
	close(r.stop)
	if err := r.conn.Close(); err != nil {
		log.Error(errors.Wrapf(err, "failed to close grpc connection"))
	}
	return nil
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

type dataMsg struct {
	rn  uint32
	rec *record.Material
}

type position struct {
	pn insolar.PulseNumber
	rn uint32
}

func (r *Replicator) currentPosition() position {
	db := r.ConnectionHolder.DB()
	rec := &raw.Record{}
	err := db.Model(rec).Last()
	if err != nil {
		log.Debug(errors.Wrapf(err, "failed to get last record from db"))
		return position{pn: 0, rn: 0}
	}
	id := insolar.NewIDFromBytes(rec.Key)
	return position{pn: id.Pulse(), rn: rec.Number}
}

func (r *Replicator) makeRequest(pos position) *exporter.GetRecords {
	return &exporter.GetRecords{
		Count:        r.cfg.Replicator.BatchSize,
		PulseNumber:  pos.pn,
		RecordNumber: pos.rn,
	}
}