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
	"context"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/observability"
)

func NewRouter(cfg *configuration.Configuration, obs *observability.Observability) *Router {
	router := httprouter.New()
	hs := &http.Server{Addr: cfg.API.Addr, Handler: router}
	r := &Router{
		hs:  hs,
		obs: obs,
	}
	router.GET("/healthcheck", r.healthCheck)
	router.GET("/metrics", r.metrics)
	return r
}

type Router struct {
	hs  *http.Server
	obs *observability.Observability
}

func (r *Router) Start() {
	log := r.obs.Log()
	go func() {
		// defer panic.Catch("component.Router")

		err := r.hs.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Error(errors.Wrapf(err, "http server ListenAndServe"))
		}
	}()
}

func (r *Router) Stop() {
	log := r.obs.Log()

	if err := r.hs.Shutdown(context.Background()); err != nil {
		log.Error(errors.Wrapf(err, "http server shutdown"))
	}
}

func (r *Router) healthCheck(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "OK")
}

func (r *Router) metrics(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	ops := promhttp.HandlerOpts{
		ErrorLog: r.obs.Log(),
	}
	handler := promhttp.HandlerFor(r.obs.Metrics(), ops)
	handler.ServeHTTP(w, req)
}
