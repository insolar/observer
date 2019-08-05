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

	"github.com/insolar/observer/internal/replica"
	"github.com/insolar/observer/internal/routing"
	log "github.com/sirupsen/logrus"
)

type Components struct {
	manager *component.Manager
}

func Prepare() *Components {
	manager := component.NewManager(nil)

	router := routing.NewRouter()
	replicator := replica.NewReplicator()

	manager.Inject(
		router,
		replicator,
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
