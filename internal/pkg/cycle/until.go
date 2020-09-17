// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package cycle

import (
	"github.com/insolar/insolar/insolar"
	"math"
	"strings"
	"time"
)

type Limit int

const (
	INFINITY Limit = math.MaxInt32
)

func UntilConnectionError(f func() error, interval time.Duration, attempts Limit, log insolar.Logger) {
	// TODO: catch external interruptions
	counter := Limit(1)
	if attempts < 1 {
		attempts = 1
	}
	for {
		err := f()
		if err != nil {
			if (!strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "EOF")) || counter >= attempts {
				panic(err)
			}
			log.Errorf("Connection error, try again (attempt %d, totalAttempts %d) %+v", counter, attempts, err)
			counter++
			time.Sleep(interval)
			continue
		}
		return
	}
}
