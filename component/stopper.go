// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package component

import (
	"github.com/pkg/errors"

	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/observability"
)

func makeStopper(obs *observability.Observability, conn *connectivity.Connectivity, router *Router) func() {
	log := obs.Log()
	return func() {
		go func() {
			err := conn.PG().Close()
			if err != nil {
				log.Error(errors.Wrapf(err, "failed to close db"))
			}
		}()

		go func() {
			err := conn.GRPC().Close()
			if err != nil {
				log.Error(errors.Wrapf(err, "failed to close db"))
			}
		}()

		go func() {
			router.Stop()
		}()
	}
}
