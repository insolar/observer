// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package component

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
)

func TestSleepManager_Count(t *testing.T) {
	mc := minimock.NewController(t)
	ctx := context.Background()

	t.Run("regular", func(t *testing.T) {
		t.Parallel()
		cfg := configuration.Default()
		defer mc.Finish()

		raw := raw{
			pulse: &observer.Pulse{
				Number: 100,
			},
			currentHeavyPN: 100,
		}
		timeExecuted := time.Second
		expectedTime := cfg.Replicator.AttemptInterval - timeExecuted

		sleepManager := NewSleepManager(cfg)
		sleepTime := sleepManager.Count(ctx, &raw, timeExecuted)
		require.Equal(t, expectedTime, sleepTime)
	})

	t.Run("fast forwarding", func(t *testing.T) {
		t.Parallel()
		cfg := configuration.Default()
		defer mc.Finish()

		raw := raw{
			pulse: &observer.Pulse{
				Number: 100,
			},
			currentHeavyPN: 200,
		}
		timeExecuted := time.Second
		expectedTime := cfg.Replicator.FastForwardInterval

		sleepManager := NewSleepManager(cfg)
		sleepTime := sleepManager.Count(ctx, &raw, timeExecuted)
		require.Equal(t, expectedTime, sleepTime)
	})

	t.Run("faster than heavy", func(t *testing.T) {
		t.Parallel()
		cfg := configuration.Default()
		defer mc.Finish()

		raw := raw{
			pulse: &observer.Pulse{
				Number: 100,
			},
			currentHeavyPN: 90,
		}
		timeExecuted := time.Second
		expectedTime := cfg.Replicator.AttemptInterval - timeExecuted

		sleepManager := NewSleepManager(cfg)
		sleepTime := sleepManager.Count(ctx, &raw, timeExecuted)
		require.Equal(t, expectedTime, sleepTime)
	})

	t.Run("nil pulse", func(t *testing.T) {
		t.Parallel()
		cfg := configuration.Default()
		defer mc.Finish()

		raw := raw{
			pulse:          nil,
			currentHeavyPN: 0,
		}
		timeExecuted := time.Second
		expectedTime := cfg.Replicator.AttemptInterval - timeExecuted
		sleepManager := NewSleepManager(cfg)
		sleepTime := sleepManager.Count(ctx, &raw, timeExecuted)
		require.Equal(t, expectedTime, sleepTime)
	})
}
