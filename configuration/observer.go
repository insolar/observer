// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package configuration

import (
	"time"

	"github.com/insolar/observer/internal/pkg/cycle"
)

type Observer struct {
	Log        Log
	DB         DB
	Replicator Replicator
}

type Log struct {
	Level        string
	Format       string
	OutputType   string
	OutputParams string
	Buffer       int
}

type DB struct {
	URL      string
	PoolSize int
	Attempts cycle.Limit
	// Interval between store in db failed attempts
	AttemptInterval time.Duration
}

type Replicator struct {
	Addr            string
	MaxTransportMsg int
	Attempts        cycle.Limit
	// Interval between fetching heavy
	AttemptInterval time.Duration
	// Using when catching up heavy on empty pulses
	FastForwardInterval time.Duration
	BatchSize           uint32
	CacheSize           int
	// Replicator's metrics, health check, etc.
	Listen string
}

func (Observer) Default() *Observer {
	return &Observer{
		Replicator: Replicator{
			Addr:                "127.0.0.1:5678",
			MaxTransportMsg:     1073741824,
			Attempts:            cycle.INFINITY,
			AttemptInterval:     10 * time.Second,
			FastForwardInterval: time.Second / 4,
			BatchSize:           2000,
			CacheSize:           10000,
			Listen:              ":8888",
		},
		DB: DB{
			URL:             "postgres://postgres@localhost/postgres?sslmode=disable",
			PoolSize:        100,
			Attempts:        5,
			AttemptInterval: 3 * time.Second,
		},
		Log: Log{
			Level:        "debug",
			Format:       "text",
			OutputType:   "stderr",
			OutputParams: "",
			Buffer:       0,
		},
	}
}
