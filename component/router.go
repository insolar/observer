// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package component

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/observability"
)

func NewRouter(cfg *configuration.Observer, obs *observability.Observability) *Router {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "OK")
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		ops := promhttp.HandlerOpts{
			ErrorLog: PromHTTPLoggerAdapter{obs.Log()},
		}
		handler := promhttp.HandlerFor(obs.Metrics(), ops)
		handler.ServeHTTP(w, req)
	})

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	hs := &http.Server{Addr: cfg.Replicator.Listen, Handler: mux}

	r := &Router{
		hs:  hs,
		obs: obs,
	}

	return r
}

type PromHTTPLoggerAdapter struct {
	insolar.Logger
}

func (o PromHTTPLoggerAdapter) Println(v ...interface{}) {
	o.Error(v)
}

type RouterInterface interface {
	Start()
	Stop()
}

type Router struct {
	hs  *http.Server
	obs *observability.Observability
}

func (r *Router) Start() {
	log := r.obs.Log()
	go func() {
		log.Debugf("starting http: %+v", r.hs)
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
