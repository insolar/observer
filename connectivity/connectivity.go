// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package connectivity

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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

			options := []grpc.DialOption{limits, grpc.WithInsecure()}
			if cfg.Replicator.Auth.Required {
				log.Info("replicator auth is required, preparing auth options")
				cp, err := x509.SystemCertPool()
				if err != nil {
					log.Fatal(errors.Wrapf(err, "failed get x509 SystemCertPool"))
				}
				httpClient := &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							RootCAs: cp,
							// nolint:gosec
							InsecureSkipVerify: cfg.Replicator.InsecureTLS,
						},
					},
					Timeout: cfg.Replicator.Auth.Timeout,
				}
				perRPCCred := grpc.WithPerRPCCredentials(newTokenCredentials(httpClient, cfg.Replicator.Auth.URL,
					cfg.Replicator.Auth.Login, cfg.Replicator.Auth.Password,
					cfg.Replicator.Auth.RefreshOffset, cfg.Replicator.InsecureTLS))

				tlsOption := grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(cp, ""))
				if cfg.Replicator.InsecureTLS {
					tlsOption = grpc.WithInsecure()
				}
				options = []grpc.DialOption{limits, tlsOption, perRPCCred}
			}

			conn, err := grpc.Dial(cfg.Replicator.Addr, options...)
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
