package beauty

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
)

type Fee struct {
	ID       uint `sql:",pk_id"`
	StartSum uint64
	FinSum   uint64
	Percent  uint
}

func (f *Fee) Dump(tx *pg.Tx) error {
	if err := tx.Insert(f); err != nil {
		return errors.Wrapf(err, "failed to insert fee")
	}
	return nil
}
