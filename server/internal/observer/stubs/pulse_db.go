package stubs

import (
	"context"
	"sync"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/pulse"
	"github.com/ugorji/go/codec"

	"github.com/insolar/observer/internal/ledger/store"
)

func NewPulses(db store.DB) pulse.Accessor {
	return &pulses{db: db}
}

type pulses struct {
	lock sync.RWMutex
	db   store.DB
}

// ForPulseNumber returns pulse for provided a pulse number. If not found, ErrNotFound will be returned.
func (s *pulses) ForPulseNumber(ctx context.Context, pn insolar.PulseNumber) (pulse insolar.Pulse, err error) {
	nd, err := s.get(pn)
	if err != nil {
		return
	}
	return nd.Pulse, nil
}

// Latest returns a latest pulse saved in DB. If not found, ErrNotFound will be returned.
func (s *pulses) Latest(ctx context.Context) (pulse insolar.Pulse, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	head, err := s.head()
	if err != nil {
		return
	}
	nd, err := s.get(head)
	if err != nil {
		return
	}
	return nd.Pulse, nil
}

func (s *pulses) head() (pn insolar.PulseNumber, err error) {

	rit := s.db.NewIterator(pulseKey(insolar.PulseNumber(0xFFFFFFFF)), true)
	defer rit.Close()

	if !rit.Next() {
		return insolar.GenesisPulse.PulseNumber, pulse.ErrNotFound
	}
	return insolar.NewPulseNumber(rit.Key()), nil
}

func (s *pulses) get(pn insolar.PulseNumber) (nd dbNode, err error) {
	buf, err := s.db.Get(pulseKey(pn))
	if err == store.ErrNotFound {
		err = pulse.ErrNotFound
		return
	}
	if err != nil {
		return
	}
	nd = deserialize(buf)
	return
}

func deserialize(buf []byte) (nd dbNode) {
	dec := codec.NewDecoderBytes(buf, &codec.CborHandle{})
	dec.MustDecode(&nd)
	return nd
}

type pulseKey insolar.PulseNumber

func (k pulseKey) Scope() store.Scope {
	return store.ScopePulse
}

func (k pulseKey) ID() []byte {
	return insolar.PulseNumber(k).Bytes()
}

type dbNode struct {
	Pulse      insolar.Pulse
	Prev, Next *insolar.PulseNumber
}
