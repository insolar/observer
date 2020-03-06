// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package dbconn

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
)

func Connect(cfg configuration.DB) (*pg.DB, error) {
	opt, err := pg.ParseURL(cfg.URL)
	if err != nil {
		// pg.ParseURL uses standard url.Parse
		// witch fills url-string with password into error.
		// So we can't use errors.Wrap here and print error above in code.
		return nil, errors.New("failed to parse cfg.DB.URL")
	}
	opt.PoolSize = cfg.PoolSize
	return pg.Connect(opt), nil
}
