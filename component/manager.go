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

package component

import (
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

type Manager struct {
	stopSignal chan bool

	cfg      *configuration.Configuration
	log      logrus.Logger
	init     func() *state
	fetch    func(*state) *raw
	beautify func(*raw) *beauty
	filter   func(*beauty) *beauty
	store    func(*beauty, *state) *observer.Statistic
	stop     func()

	router *Router
}

func Prepare() *Manager {
	cfg := configuration.Load()
	obs := observability.Make(cfg)
	conn := connectivity.Make(cfg, obs)
	router := NewRouter(cfg, obs)
	return &Manager{
		stopSignal: make(chan bool, 1),
		init:       makeInitter(cfg, obs, conn),
		log:        *obs.Log(),
		fetch:      makeFetcher(cfg, obs, conn),
		beautify:   makeBeautifier(obs),
		filter:     makeFilter(obs),
		store:      makeStorer(cfg, obs, conn),
		stop:       makeStopper(obs, conn, router),
		router:     router,
		cfg:        cfg,
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
	raw := m.fetch(s)
	beauty := m.beautify(raw)
	collapsed := m.filter(beauty)
	statistic := m.store(collapsed, s)

	sleepTime := m.cfg.Replicator.AttemptInterval

	if raw != nil {
		s.last = raw.pulse.Number
		s.rp.ShouldIterateFrom = raw.shouldIterateFrom
		// fast forward, empty pulses
		if raw.shouldIterateFrom > raw.pulse.Number {
			// todo add to config
			sleepTime = m.cfg.Replicator.FastForwardInterval
		}
	}

	if statistic != nil {
		s.stat = *statistic
	}

	// todo replace by adjustable timer
	m.log.Debug("Sleep: ", sleepTime)
	time.Sleep(sleepTime)
}

type raw struct {
	pulse             *observer.Pulse
	batch             []*observer.Record
	shouldIterateFrom insolar.PulseNumber
}

type beauty struct {
	pulse       *observer.Pulse
	records     []*observer.Record
	requests    []*observer.Request
	results     []*observer.Result
	activates   []*observer.Activate
	amends      []*observer.Amend
	deactivates []*observer.Deactivate

	transfers []*observer.ExtendedTransfer
	members   map[insolar.ID]*observer.Member
	balances  map[insolar.ID]*observer.Balance
	deposits  map[insolar.ID]*observer.Deposit
	updates   map[insolar.ID]*observer.DepositUpdate
	addresses map[string]*observer.MigrationAddress
	wastings  map[string]*observer.Wasting
}

type state struct {
	last insolar.PulseNumber
	rp   RecordPosition
	stat observer.Statistic
}

type RecordPosition struct {
	Last              insolar.PulseNumber
	RN                uint32
	ShouldIterateFrom insolar.PulseNumber
}
