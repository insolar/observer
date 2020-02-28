// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

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
