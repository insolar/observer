package observer

import (
	"context"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
)

type Record record.Material

type RecordStorage interface {
	Last() *Record
	Count(insolar.PulseNumber) uint32
	Insert(*Record) error
}

//go:generate minimock -i github.com/insolar/observer/internal/app/observer.HeavyRecordFetcher -o ./ -s _mock.go -g

type HeavyRecordFetcher interface {
	Fetch(context.Context, insolar.PulseNumber) (map[uint32]*exporter.Record, insolar.PulseNumber, error)
}

func (r *Record) Marshal() ([]byte, error) {
	return (*record.Material)(r).Marshal()
}

func (r *Record) Unmarshal(data []byte) error {
	return (*record.Material)(r).Unmarshal(data)
}
