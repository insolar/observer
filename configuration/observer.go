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
	Auth                Auth
	// Replicator's metrics, health check, etc.
	Listen string
}

type Auth struct {
	// warning: set false only for testing purpose within secured environment
	Required bool
	URL      string
	Login    string
	Password string
	// number of seconds remain of token expiration to start token refreshing
	RefreshOffset int64
	Timeout       time.Duration
	// warning: set true only for testing purpose within secured environment
	InsecureTLS bool
}

func (Observer) Default() *Observer {
	return &Observer{
		Replicator: Replicator{
			Addr:                "explorer.insolar.io:443",
			MaxTransportMsg:     1073741824,
			Attempts:            cycle.INFINITY,
			AttemptInterval:     10 * time.Second,
			FastForwardInterval: time.Second / 4,
			BatchSize:           2000,
			CacheSize:           10000,
			Auth: Auth{
				Required:      true,
				URL:           "https://wallet-api.insolar.io/auth/token",
				Login:         "${LOGIN}",
				Password:      "${PASSWORD}",
				RefreshOffset: 60,
				Timeout:       15 * time.Second,
				InsecureTLS:   false,
			},
			Listen: ":8888",
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
