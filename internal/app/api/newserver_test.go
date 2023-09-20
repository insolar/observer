// +build node

package api

import (
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
)

func NewServer(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage) ServerInterface {
	return NewObserverServer(db, log, pStorage, configuration.API{})
}
