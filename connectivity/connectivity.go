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

package connectivity

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/observability"
)

func Make(cfg *configuration.Configuration, obs *observability.Observability) *Connectivity {
	log := obs.Log()
	return &Connectivity{
		pg: func() *pg.DB {
			opt, err := pg.ParseURL(cfg.DB.URL)
			if err != nil {
				log.Fatal(errors.Wrapf(err, "failed to parse cfg.DB.URL"))
			}
			db := pg.Connect(opt)
			return db
		}(),
		grpc: func() *grpc.ClientConn {
			limits := grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(cfg.Replicator.MaxTransportMsg),
				grpc.MaxCallSendMsgSize(cfg.Replicator.MaxTransportMsg),
			)
			log.Infof("trying connect to %s...", cfg.Replicator.Addr)

			// We omit error here because connect happens in background.
			conn, err := grpc.Dial(cfg.Replicator.Addr, limits, grpc.WithInsecure())
			if err != nil {
				log.Fatal(errors.Wrapf(err, "failed to grpc.Dial"))
			}
			return conn
		}(),
	}
}

type Connectivity struct {
	pg   *pg.DB
	grpc *grpc.ClientConn
}

func (c *Connectivity) PG() *pg.DB {
	return c.pg
}

func (c *Connectivity) GRPC() *grpc.ClientConn {
	return c.grpc
}
