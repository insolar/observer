package beauty

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
)

type WasteMigrationAddress struct {
	Addr string
}

func (a *WasteMigrationAddress) Dump(tx *pg.Tx) error {
	res, err := tx.Model(&MigrationAddress{}).
		Where("addr=?", a.Addr).
		Set("wasted=true").
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update migration address")
	}

	if res.RowsAffected() != 1 {
		return errors.Errorf("failed to update migration address rows_affected=%d", res.RowsAffected())
	}

	return nil
}
