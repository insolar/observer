// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package connectivity

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
	"time"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/dbconn"
	"github.com/insolar/observer/internal/pkg/cycle"
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
							InsecureSkipVerify: cfg.Replicator.Auth.InsecureTLS,
						},
					},
					Timeout: cfg.Replicator.Auth.Timeout,
				}
				perRPCCred := grpc.WithPerRPCCredentials(newTokenCredentials(httpClient, cfg.Replicator.Auth.URL,
					cfg.Replicator.Auth.Login, cfg.Replicator.Auth.Password,
					cfg.Replicator.Auth.RefreshOffset, cfg.Replicator.Auth.InsecureTLS))

				tlsOption := grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(cp, ""))
				if cfg.Replicator.Auth.InsecureTLS {
					tlsOption = grpc.WithInsecure()
				}
				unaryRepeater := grpc.WithUnaryInterceptor(func(
					ctx context.Context,
					method string,
					req, reply interface{},
					cc *grpc.ClientConn,
					invoker grpc.UnaryInvoker,
					opts ...grpc.CallOption) (err error) {
					RetryWhileLimited(ctx, func() error {
						err = invoker(ctx, method, req, reply, cc)
						return err
					}, cfg.DB.AttemptInterval, cfg.DB.Attempts, log)
					return err
				})
				options = []grpc.DialOption{limits, tlsOption, perRPCCred, unaryRepeater}
			}

			conn, err := grpc.Dial(cfg.Replicator.Addr, options...)
			if err != nil {
				log.Fatal(errors.Wrapf(err, "failed to grpc.Dial"))
			}
			return conn
		}(),
	}
}

func RetryWhileLimited(
	ctx context.Context,
	do func() error,
	interval time.Duration,
	attempts cycle.Limit,
	log insolar.Logger,
) {
	counter := cycle.Limit(1)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := do()
		if err != nil {
			if ExporterLimited(err) && counter < attempts {
				log.WithFields(map[string]interface{}{
					"attempt":       counter,
					"attempt_limit": attempts,
				}).Errorf("Exporter rate limit exceeded. Will try again in %s", interval.String())
				time.Sleep(interval)
				counter++
				continue
			}
		}
		return
	}
}

func ExporterLimited(err error) bool {
	s := status.Convert(err)
	return s.Code() == codes.ResourceExhausted &&
		strings.Contains(s.Message(), exporter.RateLimitExceededMsg)
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
