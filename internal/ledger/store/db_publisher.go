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

package store

import (
	"context"

	"github.com/insolar/OracleMigrationToken/ins/component"
	"github.com/pkg/errors"
)

func NewDBPublisher(db DB) (DB, DBSetPublisher) {
	pub := &publisher{db: db}
	return pub, pub
}

func (p *publisher) Get(key Key) ([]byte, error) {
	return p.db.Get(key)
}

func (p *publisher) Set(key Key, value []byte) error {
	err := p.db.Set(key, value)
	if err != nil {
		return errors.Wrapf(err, "failed to save kv pair to store.DB")
	}
	p.notify(key, value)
	return nil
}
func (p *publisher) NewIterator(pivot Key, reverse bool) Iterator {
	return p.db.NewIterator(pivot, reverse)
}

func (p *publisher) Stop(ctx context.Context) error {
	if stopper, ok := p.db.(component.Stopper); ok {
		return stopper.Stop(ctx)
	}
	return nil
}

type DBSetHandle func(key Key, value []byte)

type DBSetPublisher interface {
	Subscribe(handle DBSetHandle)
}

type publisher struct {
	db       DB
	handlers []DBSetHandle
}

func (p *publisher) Subscribe(handle DBSetHandle) {
	p.handlers = append(p.handlers, handle)
}

func (p *publisher) notify(key Key, value []byte) {
	for _, h := range p.handlers {
		h(key, value)
	}
}
