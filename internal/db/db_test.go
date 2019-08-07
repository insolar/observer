package db

import (
	"context"
	"testing"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/insolar/gen"

	"github.com/stretchr/testify/require"

	raw2 "github.com/insolar/observer/internal/model/raw"
)

func TestInsertUpdate(t *testing.T) {
	ctx := context.Background()
	holder := NewConnectionHolder()
	holder.(component.Initer).Init(ctx)
	defer holder.(component.Stopper).Stop(ctx)

	db := holder.DB()

	err := db.RunInTransaction(func(tx *pg.Tx) error {
		key := gen.ID().Bytes()
		if res, err := tx.Model(&raw2.Record{}).
			Where("number=?", 42).
			Set("number=?", 43).
			Update(); err != nil {
			return err
		} else {
			t.Logf("rows=%d", res.RowsAffected())
		}
		if err := tx.Insert(&raw2.Record{Key: key, Value: []byte{1, 2, 3}, Number: 42}); err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
}
