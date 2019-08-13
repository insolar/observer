package beauty

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
)

type BalanceUpdate struct {
	ID        string
	PrevState string
	Balance   string
}

func (u *BalanceUpdate) Dump(tx *pg.Tx) error {
	res, err := tx.Model(&Member{}).
		Where("account_state=?", u.PrevState).
		Set("balance=?,account_state=?", u.Balance, u.ID).
		Update()
	if err != nil {
		return errors.Wrapf(err, "failed to update member balance")
	}
	if res.RowsAffected() != 1 {
		return errors.Errorf("failed to update member balance rows_affected=%d", res.RowsAffected())
	}
	return nil
}
