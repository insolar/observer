// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

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
	cfg *configuration.Observer
}

func NewSleepManager(cfg *configuration.Observer) *SleepManager {
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
