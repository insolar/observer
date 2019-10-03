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

package replicator

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/db"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/model/raw"
	"github.com/insolar/observer/internal/panic"
)

type DataHandle func(uint32, *record.Material)

type OnData interface {
	SubscribeOnData(handle DataHandle)
}

type DumpHandle func(ctx context.Context, tx orm.DB, pub OnDumpSuccess) error

type OnDump interface {
	SubscribeOnDump(DumpHandle)
}

type SuccessHandle func()

//go:generate minimock -i OnDumpSuccess -o ./ -s _mock.go -g

type OnDumpSuccess interface {
	Subscribe(SuccessHandle)
}

type PulseHandle func(pn insolar.PulseNumber, entropy insolar.Entropy, timestamp int64)

type OnPulse interface {
	SubscribeOnPulse(handle PulseHandle)
}

type Replicator struct {
	Configurator     configuration.Configurator `inject:""`
	ConnectionHolder db.ConnectionHolder        `inject:""`
	cfg              *configuration.Configuration

	sync.RWMutex
	dataHandles  []DataHandle
	dumpHandles  []DumpHandle
	pulseHandles []PulseHandle

	conn *grpc.ClientConn
	stop chan bool
}

func NewReplicator() *Replicator {
	return &Replicator{
		stop: make(chan bool),
	}
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

func (r *Replicator) SubscribeOnPulse(handle PulseHandle) {
	r.Lock()
	defer r.Unlock()

	r.pulseHandles = append(r.pulseHandles, handle)
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
	pulse, err := r.lastSyncedPulse()
	if err != nil {
		return err
	}
	current := r.currentPosition(ctx)
	log.Infof("Starting replication from position (rn: %d, pulse: %d)", current.rn, current.pn)
	req := r.makeRequest(current)
	go func() {
		defer panic.Log("replicator")

		for {
			batch, pos := r.pull(ctx, req)

			log.Info("emitting batch")
			r.emit(ctx, current, batch)
			log.Info("after emit")
			if len(batch) < int(r.cfg.Replicator.BatchSize) {
				if pos != current {
					r.emitDump(ctx, pos.pn)
				}
				pulse = r.syncPulses(ctx, pulse)
				time.Sleep(r.cfg.Replicator.RequestDelay)
			}
			current = pos
			req = r.makeRequest(current)

			stats.Record(ctx, lastSyncPulse.M(int64(pulse)))

			if r.needStop() {
				return
			}
		}
	}()
	return nil
}

func (r *Replicator) syncPulses(ctx context.Context, pn insolar.PulseNumber) insolar.PulseNumber {
	lastSynced := pn
	for {
		next, entropy, timestamp := r.pullPulse(lastSynced)
		if next == lastSynced {
			return next
		}
		r.emitPulse(next, entropy, timestamp)
		r.emitDump(ctx, next)

		lastSynced = next
	}
}

func (r *Replicator) pullPulse(pn insolar.PulseNumber) (insolar.PulseNumber, insolar.Entropy, int64) {
	req := &exporter.GetPulses{PulseNumber: pn, Count: 1}
	ctx := context.Background()
	client := exporter.NewPulseExporterClient(r.conn)
	stream, err := client.Export(ctx, req)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method"))
		return pn, insolar.Entropy{}, 0
	}

	log.Infof("try to pull pulse %v", req)
	resp, err := stream.Recv()
	if err == io.EOF {
		return pn, insolar.Entropy{}, 0
	}
	if err != nil {
		log.Warn(errors.Wrapf(err, "received error value from pulses gRPC stream %v", req))
		return pn, insolar.Entropy{}, 0
	}
	return resp.PulseNumber, resp.Entropy, resp.PulseTimestamp
}

func (r *Replicator) emitPulse(pn insolar.PulseNumber, entropy insolar.Entropy, timestamp int64) {
	r.RLock()
	defer r.RUnlock()

	for _, handle := range r.pulseHandles {
		handle(pn, entropy, timestamp)
	}
}

func (r *Replicator) pull(ctx context.Context, req *exporter.GetRecords) ([]dataMsg, position) {
	pulse, number := req.PulseNumber, req.RecordNumber
	client := exporter.NewRecordExporterClient(r.conn)
	stream, err := client.Export(ctx, req)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method"))
		return []dataMsg{}, position{pn: pulse, rn: number}
	}

	var batch []dataMsg
	log.Infof("try to pull records from %v", req)

	startPull := time.Now()
	defer func() {
		endPull := float64(time.Since(startPull).Nanoseconds()) / 1e6
		stats.Record(ctx,
			processingTime.M(endPull),
		)
	}()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "received error value from records gRPC stream %v", req))
			break
		}
		if resp.ShouldIterateFrom != nil {
			return batch, position{pn: *resp.ShouldIterateFrom, rn: 0}
		}
		rec, rn := &resp.Record, resp.RecordNumber
		batch = append(batch, dataMsg{rec: rec, rn: rn})
		pulse, number = rec.ID.Pulse(), rn
		log.Debugf("received %d %d %s \n", rn, pulse, rec.ID.String())
	}
	log.Info("records stream finished")

	return batch, position{pn: pulse, rn: number}
}

func (r *Replicator) emit(ctx context.Context, last position, batch []dataMsg) {
	lastPulse := last.pn
	for _, msg := range batch {
		pn := msg.rec.ID.Pulse()
		if pn != lastPulse {
			r.emitDump(ctx, lastPulse)
			lastPulse = pn
		}
		r.emitData(msg.rn, msg.rec)
		log.Debugf("emit %d %s", msg.rn, msg.rec.ID.String())
	}
}

func (r *Replicator) emitData(rn uint32, rec *record.Material) {
	r.RLock()
	defer r.RUnlock()

	for _, handle := range r.dataHandles {
		handle(rn, rec)
	}
}

func (r *Replicator) emitDump(ctx context.Context, pn insolar.PulseNumber) {
	log.Info("emit dump")

	startTime := time.Now()
	defer func() {
		endTime := float64(time.Since(startTime).Nanoseconds()) / 1e6
		stats.Record(ctx,
			processingTime.M(endTime),
		)
	}()

	db := r.ConnectionHolder.DB()
	emitter := &successEmitter{}
	for {
		err := db.RunInTransaction(func(tx *pg.Tx) error {
			for _, handle := range r.dumpHandles {
				if err := handle(ctx, tx, emitter); err != nil {
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
	log.Info("emit success tx")
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

func (r *Replicator) currentPosition(ctx context.Context) position {
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

func (r *Replicator) lastSyncedPulse() (insolar.PulseNumber, error) {
	db := r.ConnectionHolder.DB()
	count, err := db.Model(&beauty.Pulse{}).Count()
	if err != nil {
		return 0, errors.Wrapf(err, "failed request to db")
	}
	if count == 0 {
		return 0, nil
	}
	pulse := &beauty.Pulse{}
	err = db.Model(pulse).Last()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get last pulse from db")
	}
	return pulse.Pulse, nil
}

func (r *Replicator) makeRequest(pos position) *exporter.GetRecords {
	return &exporter.GetRecords{
		Count:        r.cfg.Replicator.BatchSize,
		PulseNumber:  pos.pn,
		RecordNumber: pos.rn,
	}
}
