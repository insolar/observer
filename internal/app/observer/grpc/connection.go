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

package grpc

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/insolar/observer/configuration"
)

type ConnectionHolder struct {
	conn *grpc.ClientConn
}

func NewConnectionHolder(cfg *configuration.Configuration, log *logrus.Logger) *ConnectionHolder {
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
