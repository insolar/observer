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

package replica

import (
	"context"
	"github.com/insolar/insolar/ledger/heavy/executor"
	"sync"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/ledger/store"
)

func NewJetKeeper(db store.DB) executor.JetKeeper {
	return &jetKeeper{db: db}
}

type jetKeeper struct {
	sync.RWMutex
	db store.DB
}

func (jk *jetKeeper) Add(context.Context, insolar.PulseNumber, insolar.JetID) error {
	return errors.New("not implemented JetKeeper.Add")
}

func (jk *jetKeeper) TopSyncPulse() insolar.PulseNumber {
	jk.RLock()
	defer jk.RUnlock()

	it := jk.db.NewIterator(syncPulseKey(0xFFFFFFFF), true)
	defer it.Close()
	if it.Next() {
		return insolar.NewPulseNumber(it.Key()[1:])
	}
	return insolar.GenesisPulse.PulseNumber
}

func (jk *jetKeeper) Update(sync insolar.PulseNumber) error {
	jk.Lock()
	defer jk.Unlock()

	return jk.updateSyncPulse(sync)
}

func (jk *jetKeeper) Subscribe(at insolar.PulseNumber, handler func(insolar.PulseNumber)) {
	inslogger.FromContext(context.Background()).Errorf("not implmented JetKeeper.Subscribe")
}

func (jk *jetKeeper) updateSyncPulse(pn insolar.PulseNumber) error {
	err := jk.db.Set(syncPulseKey(pn), []byte{})
	if err != nil {
		return errors.Wrapf(err, "failed to set up new sync pulse")
	}
	return nil
}

type syncPulseKey insolar.PulseNumber

func (k syncPulseKey) Scope() store.Scope {
	return store.ScopeJetKeeper
}

func (k syncPulseKey) ID() []byte {
	return append([]byte{0x02}, insolar.PulseNumber(k).Bytes()...)
}
