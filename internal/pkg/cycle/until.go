// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package cycle

import (
	"math"
	"time"
)

type Limit int

const (
	INFINITY Limit = math.MaxInt32
)

func UntilError(f func() error, interval time.Duration, attempts Limit) {
	// TODO: catch external interruptions
	counter := Limit(1)
	if attempts < 1 {
		attempts = 1
	}
	for {
		err := f()
		if err != nil {
			if counter >= attempts {
				panic(err)
			}
			counter++
			time.Sleep(interval)
			continue
		}
		return
	}
}
