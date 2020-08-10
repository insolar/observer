// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package component

import (
	"context"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/grpc"
	"github.com/insolar/observer/observability"
)

type Manager struct {
	stopSignal chan bool

	cfg           *configuration.Observer
	log           insolar.Logger
	init          func() *state
	commonMetrics *observability.CommonObserverMetrics
	fetch         func(context.Context, *state) *raw
	beautify      func(context.Context, *raw) *beauty
	filter        func(*beauty) *beauty
	store         func(*beauty, *state) (*observer.Statistic, error)
	stop          func()

	router       RouterInterface
	sleepCounter sleepCounter
}

func Prepare(ctx context.Context, cfg *configuration.Observer) *Manager {
	obs := observability.Make(ctx)
	conn := connectivity.Make(cfg, obs)
	router := NewRouter(cfg, obs)
	pulses := grpc.NewPulseFetcher(cfg, obs, exporter.NewPulseExporterClient(conn.GRPC()))
	records := grpc.NewRecordFetcher(cfg, obs, exporter.NewRecordExporterClient(conn.GRPC()))
	sm := NewSleepManager(cfg)
	return &Manager{
		stopSignal:    make(chan bool, 1),
		init:          makeInitter(cfg, obs, conn),
		log:           obs.Log(),
		commonMetrics: observability.MakeCommonMetrics(obs),
		fetch:         makeFetcher(obs, pulses, records),
		beautify:      makeBeautifier(cfg, obs, conn),
		filter:        makeFilter(obs),
		store:         makeStorer(cfg, obs, conn),
		stop:          makeStopper(obs, conn, router),
		router:        router,
		cfg:           cfg,
		sleepCounter:  sm,
	}
}

func (m *Manager) Start() {
	go func() {
		m.router.Start()
		defer m.stop()

		state := m.init()
		for {
			m.run(state)
			if m.needStop() {
				return
			}
		}
	}()
}

func (m *Manager) Stop() {
	m.stopSignal <- true
}

func (m *Manager) needStop() bool {
	select {
	case <-m.stopSignal:
		return true
	default:
		// continue
	}
	return false
}

func (m *Manager) run(s *state) {
	timeStart := time.Now()
	m.log.Debug("Timer: new round started at ", timeStart)
	ctx := context.Background()

	tempTimer := time.Now()
	raw := m.fetch(ctx, s)
	m.log.Debug("Timer: fetched ", time.Since(tempTimer))

	tempTimer = time.Now()
	beauty := m.beautify(ctx, raw)
	m.log.Debug("Timer: beautified ", time.Since(tempTimer))

	tempTimer = time.Now()
	collapsed := m.filter(beauty)
	m.log.Debug("Timer: filtered ", time.Since(tempTimer))

	tempTimer = time.Now()
	statistic := m.storeWithRetries(s, collapsed)
	m.log.Debug("Timer: stored ", time.Since(tempTimer))

	timeExecuted := time.Since(timeStart)
	m.commonMetrics.PulseProcessingTime.Set(timeExecuted.Seconds())
	m.log.Debug("Timer: executed ", timeExecuted)
	m.log.Debugf("Stats: %+v", statistic)

	if raw != nil {
		s.last = raw.pulse.Number
		s.ShouldIterateFrom = raw.shouldIterateFrom
	}

	sleepTime := m.sleepCounter.Count(ctx, raw, timeExecuted)
	m.log.Info("Sleep: ", sleepTime)
	time.Sleep(sleepTime)
}

func (m *Manager) storeWithRetries(s *state, collapsed *beauty) *observer.Statistic {
	var statistic *observer.Statistic
	for i := 1; i <= m.cfg.DB.Retries; i++ {
		var err error
		statistic, err = m.store(collapsed, s)
		if err == nil {
			break
		}
		m.log.Errorf("DB connection error... %s", err.Error())
		if i == m.cfg.DB.Retries {
			panic(err)
		}
		time.Sleep(m.cfg.DB.Delay * time.Duration(i))
	}
	return statistic
}

type raw struct {
	pulse             *observer.Pulse
	batch             map[uint32]*exporter.Record
	shouldIterateFrom insolar.PulseNumber
	currentHeavyPN    insolar.PulseNumber
}

type beauty struct {
	pulse       *observer.Pulse
	requests    []*observer.Request
	results     []*observer.Result
	activates   []*observer.Activate
	amends      []*observer.Amend
	deactivates []*observer.Deactivate

	members        map[insolar.ID]*observer.Member
	balances       map[insolar.ID]*observer.Balance
	deposits       map[insolar.ID]observer.Deposit
	depositUpdates map[insolar.ID]observer.DepositUpdate
	depositMembers map[insolar.Reference]observer.DepositMemberUpdate
	addresses      map[string]*observer.MigrationAddress
	vestings       map[string]*observer.Vesting

	txRegister         []observer.TxRegister
	txResult           []observer.TxResult
	txSagaResult       []observer.TxSagaResult
	txDepositTransfers []observer.TxDepositTransferUpdate
}

type state struct {
	last              insolar.PulseNumber
	ShouldIterateFrom insolar.PulseNumber
	currentHeavyPN    insolar.PulseNumber
	ms                metricState
}

type RecordPosition struct {
	ShouldIterateFrom insolar.PulseNumber
}
