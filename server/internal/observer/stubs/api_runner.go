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

package stubs

import (
	"context"
	"crypto"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/insolar/rpc/v2"
	"github.com/pkg/errors"

	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
)

// NewRunner is C-tor for API Runner.
func NewRunner(cfg *configuration.APIRunner, cfgTransport *configuration.Transport) (insolar.APIRunner, error) {

	if err := checkConfig(cfg); err != nil {
		return nil, errors.Wrap(err, "[ NewAPIRunner ] Bad APIRunner config")
	}

	if cfgTransport == nil {
		return nil, errors.New("[ NewAPIRunner ] Bad Transport config")
	}
	addrStr := fmt.Sprint(cfg.Address)
	rpcServer := rpc.NewServer()
	ar := apiRunnerStub{
		server:    &http.Server{Addr: addrStr},
		rpcServer: rpcServer,
		cfg:       cfg,
		timeout:   30 * time.Second,
		keyCache:  make(map[string]crypto.PublicKey),
		cacheLock: &sync.RWMutex{},
	}

	return &ar, nil
}

type apiRunnerStub struct {
	server    *http.Server
	rpcServer *rpc.Server
	cfg       *configuration.APIRunner
	keyCache  map[string]crypto.PublicKey
	cacheLock *sync.RWMutex
	timeout   time.Duration
}

// IsAPIRunner is implementation of APIRunner interface for component manager.
func (ar *apiRunnerStub) IsAPIRunner() bool {
	return true
}

// Start runs api server.
func (ar *apiRunnerStub) Start(ctx context.Context) error {

	router := http.NewServeMux()
	ar.server.Handler = router

	router.HandleFunc("/healthcheck", ar.healthCheck)
	router.Handle(ar.cfg.RPC, ar.rpcServer)

	inslog := inslogger.FromContext(ctx)
	inslog.Info("Starting Observer ApiRunner ...")
	inslog.Info("Config: ", ar.cfg)
	listener, err := net.Listen("tcp", ar.server.Addr)
	if err != nil {
		return errors.Wrap(err, "Can't start listening")
	}
	go func() {
		if err := ar.server.Serve(listener); err != http.ErrServerClosed {
			inslog.Error("Http server: ListenAndServe() error: ", err)
		}
	}()
	return nil
}

// Stop stops api server.
func (ar *apiRunnerStub) Stop(ctx context.Context) error {
	const timeOut = 5

	inslogger.FromContext(ctx).Infof("Shutting down server gracefully ...(waiting for %d seconds)", timeOut)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeOut)*time.Second)
	defer cancel()
	err := ar.server.Shutdown(ctxWithTimeout)
	if err != nil {
		return errors.Wrap(err, "Can't gracefully stop API server")
	}

	return nil
}

func checkConfig(cfg *configuration.APIRunner) error {
	if cfg == nil {
		return errors.New("[ checkConfig ] config is nil")
	}
	if cfg.Address == "" {
		return errors.New("[ checkConfig ] Address must not be empty")
	}
	if len(cfg.Call) == 0 {
		return errors.New("[ checkConfig ] Call must exist")
	}
	if len(cfg.RPC) == 0 {
		return errors.New("[ checkConfig ] RPC must exist")
	}

	return nil
}

func (ar *apiRunnerStub) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
