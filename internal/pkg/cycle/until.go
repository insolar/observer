package cycle

import (
	"math"
	"strings"
	"time"

	"github.com/insolar/insolar/insolar"
)

type Limit int

const (
	INFINITY Limit = math.MaxInt32
)

func UntilConnectionError(f func() error, interval time.Duration, attempts Limit, log insolar.Logger) {
	condition := func(err error) bool {
		return !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "EOF")
	}
	untilError(f, condition, interval, attempts, log)
}

func UntilError(f func() error, interval time.Duration, attempts Limit, log insolar.Logger) {
	untilError(f, nil, interval, attempts, log)
}

func untilError(f func() error, condition func(err error) bool, interval time.Duration, attempts Limit, log insolar.Logger) {
	// TODO: catch external interruptions
	counter := Limit(1)
	if attempts < 1 {
		attempts = 1
	}
	for {
		err := f()
		if err != nil {
			if (condition != nil && condition(err)) || counter >= attempts {
				panic(err)
			}
			log.Errorf("error, try again (attempt %d, totalAttempts %d) %+v", counter, attempts, err)
			counter++
			time.Sleep(interval)
			continue
		}
		return
	}
}
