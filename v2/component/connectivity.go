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
	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/internal/app/observer/grpc"
	"github.com/insolar/observer/v2/internal/app/observer/postgres"
)

func makeConnectivity(cfg *configuration.Configuration, obs *Observability) *Connectivity {
	log := obs.Log()
	return &Connectivity{
		pg:   postgres.NewConnectionHolder(cfg),
		grpc: grpc.NewConnectionHolder(cfg, log),
	}
}

type Connectivity struct {
	pg   *postgres.ConnectionHolder
	grpc *grpc.ConnectionHolder
}

func (c *Connectivity) PG() *postgres.ConnectionHolder {
	return c.pg
}

func (c *Connectivity) GRPC() *grpc.ConnectionHolder {
	return c.grpc
}
