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
	"context"
	"time"

	"github.com/insolar/observer/configuration"
)

type sleepCounter interface {
	Count(ctx context.Context, raw *raw, timeExecuted time.Duration) time.Duration
}

type SleepManager struct {
	cfg *configuration.Configuration
}

func NewSleepManager(cfg *configuration.Configuration) *SleepManager {
	return &SleepManager{
		cfg: cfg,
	}
}

func (sm *SleepManager) Count(ctx context.Context, raw *raw, timeExecuted time.Duration) time.Duration {
	if raw == nil {
		return sm.cfg.Replicator.AttemptInterval
	}

	// fast forward, empty pulses
	if raw.pulse != nil && raw.currentHeavyPN > raw.pulse.Number {
		return sm.cfg.Replicator.FastForwardInterval
	}

	// reducing sleep time by execution time
	sleepTime := sm.cfg.Replicator.AttemptInterval - timeExecuted
	return sleepTime
}
