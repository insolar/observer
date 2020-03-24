// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package connectivity

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/dbconn"
	"github.com/insolar/observer/observability"
)

func Make(cfg *configuration.Observer, obs *observability.Observability) *Connectivity {
	log := obs.Log()
	return &Connectivity{
		pg: func() *pg.DB {
			db, err := dbconn.Connect(cfg.DB)
			if err != nil {
				log.Fatal(err.Error())
			}
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
