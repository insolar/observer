// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package grpc

import (
	"google.golang.org/grpc"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/configuration"
)

type ConnectionHolder struct {
	conn *grpc.ClientConn
}

func NewConnectionHolder(cfg *configuration.Observer, log insolar.Logger) *ConnectionHolder {
	limits := grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(cfg.Replicator.MaxTransportMsg),
		grpc.MaxCallSendMsgSize(cfg.Replicator.MaxTransportMsg),
	)
	log.Infof("trying connect to %s...", cfg.Replicator.Addr)

	// We omit error here because connect happens in background.
	conn, _ := grpc.Dial(cfg.Replicator.Addr, limits, grpc.WithInsecure())
	return &ConnectionHolder{
		conn: conn,
	}
}

func (h *ConnectionHolder) Conn() *grpc.ClientConn {
	return h.conn
}
