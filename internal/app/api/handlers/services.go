// +build !node

package handlers

import (
	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/api"
	"github.com/insolar/observer/internal/app/api/services"
	"github.com/insolar/observer/internal/app/observer"
)

func RegisterHandlers(router runtime.EchoRouter, db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage, config configuration.APIConfig) {
	externalObserverAPI := api.NewObserverServerExtended(db, log, pStorage, config)
	api.RegisterHandlers(router, externalObserverAPI)

	internalObserverAPI := services.NewObserverServer(db, log, pStorage, config)
	services.RegisterHandlers(router, internalObserverAPI)
}
