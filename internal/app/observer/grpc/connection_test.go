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
	"fmt"
	"net"
	"testing"

	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/insolar/observer/configuration"
)

func getAvailableListener() (int, net.Listener, error) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, nil, err
	}
	addr, ok := lis.Addr().(*net.TCPAddr)
	if !ok {
		return 0, nil, errors.Errorf("failed to cast %T to *net.TCPAddr", lis.Addr())
	}
	return addr.Port, lis, nil
}

func startServer() (int, *grpc.Server, error) {
	port, lis, err := getAvailableListener()
	if err != nil {
		return 0, nil, errors.New("failed to open port")
	}
	server := grpc.NewServer()
	go func() {
		err = server.Serve(lis)
	}()
	fmt.Printf("grpc server started on %d port\n", port)
	return port, server, err
}

func stopServer(server *grpc.Server) {
	server.GracefulStop()
}

func TestConnectionHolder_Conn(t *testing.T) {
	log := inslogger.FromContext(inslogger.TestContext(t))

	t.Run("ordinary", func(t *testing.T) {
		cfg := configuration.Default()

		holder := NewConnectionHolder(cfg, log)
		require.NotNil(t, holder)
		require.NotNil(t, holder.Conn())
	})
}
