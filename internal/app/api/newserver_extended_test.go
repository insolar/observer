// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package api

import (
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
)

func NewServer(db *pg.DB, log insolar.Logger, pStorage observer.PulseStorage) ServerInterface {
	return NewObserverServerExtended(db, log, pStorage, configuration.APIExtended{
		FeeAmount: testFee,
		Price:     testPrice,
	})
}
