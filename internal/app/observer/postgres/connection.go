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

package postgres

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/configuration"
)

type ConnectionHolder struct {
	db *pg.DB
}

func NewConnectionHolder(cfg *configuration.Configuration) *ConnectionHolder {
	opt, err := pg.ParseURL(cfg.DB.URL)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to parse cfg.DB.URL"))
		return nil
	}
	db := pg.Connect(opt)
	return &ConnectionHolder{
		db: db,
	}
}

func (h *ConnectionHolder) DB() *pg.DB {
	return h.db
}

func (h *ConnectionHolder) Close() error {
	if err := h.db.Close(); err != nil {
		log.Error(errors.Wrapf(err, "failed to close db"))
	}
	return nil
}