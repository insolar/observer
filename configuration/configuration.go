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

package configuration

import (
	"time"

	"github.com/insolar/observer/internal/pkg/cycle"
)

type Configuration struct {
	Replicator Replicator
	DB         DB
	Log        Log
}

func (c Configuration) GetConfig() interface{} {
	return &c
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

type DB struct {
	URL      string
	PoolSize int
	Attempts cycle.Limit
	// Interval between store in db failed attempts
	AttemptInterval time.Duration
}

type Log struct {
	Level        string
	Format       string
	OutputType   string
	OutputParams string
	Buffer       int
}

func Default() *Configuration {
	return &Configuration{
		Replicator: Replicator{
			Addr:                "127.0.0.1:5678",
			MaxTransportMsg:     1073741824,
			Attempts:            cycle.INFINITY,
			AttemptInterval:     10 * time.Second,
			FastForwardInterval: time.Second / 4,
			BatchSize:           2000,
			CacheSize:           10000,
			Listen:              ":0",
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
