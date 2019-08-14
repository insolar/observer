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

package components

import (
	"context"

	"github.com/insolar/insolar/component"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/api"
	"github.com/insolar/observer/internal/beauty"
	"github.com/insolar/observer/internal/beauty/pulse"
	"github.com/insolar/observer/internal/configuration"
	"github.com/insolar/observer/internal/db"
	"github.com/insolar/observer/internal/raw"
	"github.com/insolar/observer/internal/replication"
)

type Components struct {
	manager *component.Manager
}

func Prepare() *Components {
	manager := component.NewManager(nil)

	manager.Inject(
		configuration.Load(),
		db.NewConnectionHolder(),
		replication.NewReplicator(),
		raw.NewDumper(),
		beauty.NewBeautifier(),
		pulse.NewKeeper(),
		api.NewRouter(),
	)

	return &Components{manager: manager}
}

func (c *Components) Start() {
	ctx := context.Background()
	if err := c.manager.Init(ctx); err != nil {
		log.Error(err)
	}

	if err := c.manager.Start(ctx); err != nil {
		log.Error(err)
	}
}

func (c *Components) Stop() {
	ctx := context.Background()
	if err := c.manager.Stop(ctx); err != nil {
		log.Error(err)
	}
}
