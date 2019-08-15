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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/db"
	"github.com/insolar/observer/internal/model/beauty"
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

type PulseHandle func(pn insolar.PulseNumber, entropy insolar.Entropy, timestamp int64)

type OnPulse interface {
	SubscribeOnPulse(handle PulseHandle)
}

type Replicator struct {
	Configurator     configuration.Configurator `inject:""`
	ConnectionHolder db.ConnectionHolder        `inject:""`
	cfg              *configuration.Configuration

	highestSyncPulseCollector prometheus.Gauge
	processingTime            prometheus.Summary

	sync.RWMutex
	dataHandles  []DataHandle
	dumpHandles  []DumpHandle
	pulseHandles []PulseHandle

	conn *grpc.ClientConn
	stop chan bool
}

func NewReplicator() *Replicator {
	collector := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "observer_last_sync_pulse",
		Help: "The last number that was replicated from HME.",
	})
	processingTime := promauto.NewSummary(prometheus.SummaryOpts{
		Name: "observer_processing_duration_seconds",
		Help: "The time that needs to replicate and beautify data.",
	})
	return &Replicator{
		highestSyncPulseCollector: collector,
		processingTime:            processingTime,
		stop:                      make(chan bool),
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
	current := r.currentPosition()
	log.Infof("Starting replication from position (rn: %d, pulse: %d)", current.rn, current.pn)
	req := r.makeRequest(current)
	go func() {
		defer catchPanic()

		startTime := time.Now()
		for {
			batch, pos := r.pull(req)
			log.Infof("emitting batch")
			r.emit(&startTime, current, batch)
			log.Infof("after emit")
			if len(batch) < int(r.cfg.Replicator.BatchSize) {
				if pos != current {

					r.emitDump(pos.pn)
					r.updateProcessingTime(startTime)
				}
				r.syncPulses()
				time.Sleep(r.cfg.Replicator.RequestDelay)
				startTime = time.Now()
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

func (r *Replicator) syncPulses() {
	lastSynced := r.lastSyncedPulse()
	for {
		next, entropy, timestamp := r.pullPulse(lastSynced)
		if next == lastSynced {
			return
		}
		r.emitPulse(next, entropy, timestamp)
		r.emitDump(next)
		r.updateStat(next)
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

func (r *Replicator) pull(req *exporter.GetRecords) ([]dataMsg, position) {
	ctx := context.Background()
	pulse, number := req.PulseNumber, req.RecordNumber
	client := exporter.NewRecordExporterClient(r.conn)
	stream, err := client.Export(ctx, req)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to get gRPC stream from exporter.Export method"))
		return []dataMsg{}, position{pn: pulse, rn: number}
	}

	batch := []dataMsg{}
	log.Infof("try to pull records from %v", req)
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "received error value from records gRPC stream %v", req))
			break
		}
		rec, rn := &resp.Record, resp.RecordNumber
		batch = append(batch, dataMsg{rec: rec, rn: rn})
		pulse, number = rec.ID.Pulse(), rn
		log.Infof("received %d %d %s \n", rn, pulse, rec.ID.String())
	}
	log.Infof("records stream finished")
	return batch, position{pn: pulse, rn: number}
}

func (r *Replicator) emit(start *time.Time, last position, batch []dataMsg) {
	lastPulse := last.pn
	for _, msg := range batch {
		pn := msg.rec.ID.Pulse()
		if pn != lastPulse {
			r.emitDump(lastPulse)
			lastPulse = pn
			r.updateProcessingTime(*start)
			*start = time.Now()
		}
		r.emitData(msg.rn, msg.rec)
		log.Infof("emit %d %s", msg.rn, msg.rec.ID.String())
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
	log.Infof("emit dump")
	db := r.ConnectionHolder.DB()
	emitter := &successEmitter{}
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
	log.Infof("emit success tx")
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
	r.setStat(id.Pulse())
	return position{pn: id.Pulse(), rn: rec.Number}
}

func (r *Replicator) lastSyncedPulse() insolar.PulseNumber {
	db := r.ConnectionHolder.DB()
	pulse := &beauty.Pulse{}
	err := db.Model(pulse).Last()
	if err != nil {
		log.Debug(errors.Wrapf(err, "failed to get last pulse from db"))
		return 0
	}
	return pulse.Pulse
}

func (r *Replicator) makeRequest(pos position) *exporter.GetRecords {
	return &exporter.GetRecords{
		Count:        r.cfg.Replicator.BatchSize,
		PulseNumber:  pos.pn,
		RecordNumber: pos.rn,
	}
}

func (r *Replicator) setStat(pn insolar.PulseNumber) {
	r.highestSyncPulseCollector.Set(float64(pn))
}

func (r *Replicator) updateStat(pn insolar.PulseNumber) {
	r.highestSyncPulseCollector.Set(float64(pn))
}

func (r *Replicator) updateProcessingTime(start time.Time) {
	diff := time.Since(start)
	r.processingTime.Observe(diff.Seconds())
}

func catchPanic() {
	if err := recover(); err != nil {
		log.Error(err)
	}
}
