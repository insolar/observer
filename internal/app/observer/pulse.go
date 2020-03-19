// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package observer

import (
	"context"

	"github.com/insolar/insolar/insolar"
)

type Pulse struct {
	Number    insolar.PulseNumber
	Entropy   insolar.Entropy
	Timestamp int64
	Nodes     []insolar.Node
}

type PulseStorage interface {
	Insert(*Pulse) error
	Last() (*Pulse, error)
	GetRange(fromTimestamp, toTimestamp int64, limit int, pulseNumber *int64) ([]uint32, error)
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer.PulseFetcher -o ./ -s _mock.go -g
type PulseFetcher interface {
	Fetch(context.Context, insolar.PulseNumber) (*Pulse, error)
	FetchCurrent(ctx context.Context) (insolar.PulseNumber, error)
}
