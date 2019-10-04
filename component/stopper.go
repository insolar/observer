//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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
