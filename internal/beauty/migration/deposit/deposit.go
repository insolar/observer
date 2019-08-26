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

package deposit

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/deposit"
	depositProxy "github.com/insolar/insolar/logicrunner/builtin/proxy/deposit"
	"github.com/insolar/insolar/pulse"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/insolar/observer/internal/dto"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replication"

	log "github.com/sirupsen/logrus"
)

type depositBuilder struct {
	res *record.Material
	act *record.Material
}

func (b *depositBuilder) String() string {
	return fmt.Sprintf("res: %v act: %v", b.res, b.act)
}

func (b *depositBuilder) build() (*beauty.Deposit, error) {
	callResult := parseMemberRef(b.res)
	if callResult.status != dto.SUCCESS {
		return nil, errors.New("invalid create deposit result payload")
	}
	id := b.act.ID
	act := b.act.Virtual.GetActivate()
	deposit := initialDepositState(act)
	transferDate, err := pulse.Number(b.act.ID.Pulse()).AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deposit create pulse (%d) to time", b.act.ID.Pulse())
	}

	return &beauty.Deposit{
		EthHash:         strings.ToLower(deposit.TxHash),
		DepositRef:      act.Request.String(),
		MemberRef:       callResult.memberRef,
		TransferDate:    transferDate.Unix(),
		HoldReleaseDate: 0,
		Amount:          deposit.Amount,
		Balance:         deposit.Balance,
		DepositState:    id.String(),
	}, nil
}

type Composer struct {
	requests  map[insolar.ID]*record.Material
	results   map[insolar.ID]*record.Material
	activates map[insolar.ID]*record.Material
	builders  map[insolar.ID]*depositBuilder

	sync.RWMutex
	cache []*beauty.Deposit

	stat *dumpStat
}

func NewComposer() *Composer {
	stat := &dumpStat{
		cached: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "observer_deposit_composer_cached_total",
			Help: "Cache size of migration address composer",
		}),
	}

	return &Composer{
		requests:  make(map[insolar.ID]*record.Material),
		results:   make(map[insolar.ID]*record.Material),
		activates: make(map[insolar.ID]*record.Material),
		builders:  make(map[insolar.ID]*depositBuilder),
		stat:      stat,
	}
}

func (c *Composer) Process(rec *record.Material) {
	defer panic.Log("deposit_composer")

	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Result:
		origin := *v.Result.Request.Record()
		if req, ok := c.requests[origin]; ok {
			delete(c.requests, origin)
			request := (*dto.Request)(req)
			if request.IsIncoming() {
				switch {
				case isDepositMigrationCall(req):
					log.Infof("deposit.migration Call")
					c.processResult(rec)
				case isDepositNew(req):
					log.Infof("deposit.New")
					c.processDepositNew(req)
				}
			}
		} else {
			c.results[origin] = rec
		}
	case *record.Virtual_IncomingRequest:
		id := rec.ID
		if res, ok := c.results[id]; ok {
			delete(c.results, id)
			switch {
			case isDepositMigrationCall(rec):
				log.Infof("deposit.migration Call")
				c.processResult(res)
			case isDepositNew(rec):
				log.Infof("deposit.New")
				c.processDepositNew(rec)
			}
		} else {
			c.requests[id] = rec
		}
	case *record.Virtual_OutgoingRequest:
		id := rec.ID
		if _, ok := c.results[id]; ok {
			delete(c.results, id)
		} else {
			c.requests[id] = rec
		}
	case *record.Virtual_Activate:
		if isDepositActivate(v.Activate) {
			log.Infof("deposit.Activate")
			c.processDepositActivate(rec)
		}
	}
}

func (c *Composer) processResult(res *record.Material) {
	origin := *res.Virtual.GetResult().Request.Record()
	if b, ok := c.builders[origin]; ok {
		b.res = res
		c.compose(b)
	} else {
		c.builders[origin] = &depositBuilder{res: res}
	}
}

func (c *Composer) processDepositNew(req *record.Material) {
	direct := req.ID
	if act, ok := c.activates[direct]; ok {
		origin := *req.Virtual.GetIncomingRequest().Reason.Record()
		if b, ok := c.builders[origin]; ok {
			b.act = act
			c.compose(b)
		} else {
			c.builders[origin] = &depositBuilder{act: act}
		}
	} else {
		c.requests[direct] = req
	}
}

func (c *Composer) processDepositActivate(rec *record.Material) {
	direct := *rec.Virtual.GetActivate().Request.Record()
	if req, ok := c.requests[direct]; ok {
		origin := *req.Virtual.GetIncomingRequest().Reason.Record()
		if b, ok := c.builders[origin]; ok {
			b.act = rec
			c.compose(b)
		} else {
			c.builders[origin] = &depositBuilder{act: rec}
		}
	} else {
		c.activates[direct] = rec
	}
}

func (c *Composer) compose(b *depositBuilder) {
	c.Lock()
	defer c.Unlock()

	deposit, err := b.build()
	if err == nil {
		c.cache = append(c.cache, deposit)
	} else {
		log.Error(err)
	}

	direct := *b.act.Virtual.GetActivate().Request.Record()
	origin := *b.res.Virtual.GetResult().Request.Record()
	delete(c.activates, direct)
	delete(c.requests, direct)
	delete(c.builders, origin)
}

func (c *Composer) Dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	log.Infof("dump deposits")

	for _, dep := range c.cache {
		if err := dep.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump deposits")
		}
	}

	pub.Subscribe(func() {
		c.Lock()
		defer c.Unlock()
		c.cache = []*beauty.Deposit{}
	})
	return nil
}

func initialDepositState(act *record.Activate) *deposit.Deposit {
	d := deposit.Deposit{}
	err := insolar.Deserialize(act.Memory, &d)
	if err != nil {
		log.Error(errors.New("failed to deserialize deposit contract state"))
	}
	return &d
}

func isDepositMigrationCall(rec *record.Material) bool {
	request := (*dto.Request)(rec)
	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "deposit.migration"
}

func isDepositNew(req *record.Material) bool {
	v, ok := req.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		return false
	}

	in := v.IncomingRequest
	if in.Method != "New" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*depositProxy.PrototypeReference)
}

func isDepositActivate(act *record.Activate) bool {
	return act.Image.Equal(*depositProxy.PrototypeReference)
}

type dumpStat struct {
	cached prometheus.Gauge
}

func (c *Composer) updateStat() {
	requestCount := len(c.requests)
	resultCount := len(c.results)
	activatesCount := len(c.activates)
	buildersCount := len(c.builders)

	c.stat.cached.Set(float64(requestCount + resultCount + activatesCount + buildersCount))
}
