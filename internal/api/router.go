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

package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/configuration"
)

func NewRouter() *Router {
	return &Router{}
}

type Router struct {
	Configurator configuration.Configurator `inject:""`
	cfg          *configuration.Configuration
	hs           *http.Server
}

func (r *Router) Init(ctx context.Context) error {
	if r.Configurator != nil {
		r.cfg = r.Configurator.Actual()
	} else {
		r.cfg = configuration.Default()
	}
	router := httprouter.New()
	router.GET("/healthcheck", healthCheck)
	r.hs = &http.Server{Addr: r.cfg.API.Addr, Handler: router}
	return nil
}

func (r *Router) Start(ctx context.Context) error {
	go func() {
		err := r.hs.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Error(errors.Wrapf(err, "http server ListenAndServe"))
		}
	}()
	return nil
}

func (r *Router) Stop(ctx context.Context) error {
	if err := r.hs.Shutdown(context.Background()); err != nil {
		log.Error(errors.Wrapf(err, "http server shutdown"))
	}
	return nil
}

func healthCheck(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "OK")
}
