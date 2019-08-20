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

package member

import (
	"sync"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/beauty/member/wallet/account"
	"github.com/insolar/observer/internal/dto"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replication"
)

type memberBuilder struct {
	act *record.Material
	res *record.Material
}

func (b *memberBuilder) build() (*beauty.Member, error) {
	if b.res == nil || b.act == nil {
		return nil, errors.New("trying to create member from noncomplete builder")
	}
	if b.res.Virtual.GetResult().Payload == nil {
		return nil, errors.New("member creation result payload is nil")
	}
	params := memberStatus(b.res)
	balance := account.AccountBalance(b.act)
	ref, err := insolar.NewReferenceFromBase58(params.reference)
	if err != nil {
		return nil, errors.New("invalid member reference")
	}
	return &beauty.Member{
		MemberRef:        ref.String(),
		Balance:          balance,
		MigrationAddress: params.migrationAddress,
		AccountState:     b.act.ID.String(),
		Status:           string(params.status),
	}, nil
}

func NewComposer() *Composer {
	stat := &dumpStat{
		cached: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "observer_member_composer_cached_total",
			Help: "Cache size of migration address composer",
		}),
	}
	return &Composer{
		builders:  make(map[insolar.ID]*memberBuilder),
		requests:  make(map[insolar.ID]*record.Material),
		activates: make(map[insolar.ID]*record.Material),
		results:   make(map[insolar.ID]*record.Material),
		stat:      stat,
	}
}

type Composer struct {
	builders  map[insolar.ID]*memberBuilder
	requests  map[insolar.ID]*record.Material
	activates map[insolar.ID]*record.Material
	results   map[insolar.ID]*record.Material

	sync.RWMutex
	cache []*beauty.Member

	stat *dumpStat
}

func (c *Composer) Process(rec *record.Material) {
	defer panic.Log("member_composer")

	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Result:
		origin := *v.Result.Request.Record()
		if req, ok := c.requests[origin]; ok {
			delete(c.requests, origin)
			request := (*dto.Request)(req)
			if request.IsIncoming() {
				switch {
				case isMemberCreateRequest(req):
					c.memberCreateResult(rec)
				case account.IsNewAccount(req):
					c.newAccount(req)
				}
			}
		} else {
			c.results[origin] = rec
		}
	case *record.Virtual_IncomingRequest:
		if res, ok := c.results[rec.ID]; ok {
			delete(c.results, rec.ID)
			switch {
			case isMemberCreateRequest(rec):
				c.memberCreateResult(res)
			case account.IsNewAccount(rec):
				c.newAccount(rec)
			}
		} else {
			c.requests[rec.ID] = rec
		}
	case *record.Virtual_OutgoingRequest:
		if _, ok := c.results[rec.ID]; ok {
			delete(c.results, rec.ID)
		} else {
			c.requests[rec.ID] = rec
		}
	case *record.Virtual_Activate:
		if account.IsAccountActivate(v.Activate) {
			c.accountActivate(rec)
		}
	}
}

func (c *Composer) memberCreateResult(rec *record.Material) {
	log.Infof("member result")
	origin := *rec.Virtual.GetResult().Request.Record()
	if b, ok := c.builders[origin]; ok {
		b.res = rec
		c.compose(b)
	} else {
		c.builders[origin] = &memberBuilder{res: rec}
	}
}

func (c *Composer) newAccount(rec *record.Material) {
	log.Infof("new account")
	direct := rec.ID
	if act, ok := c.activates[direct]; ok {
		origin := *rec.Virtual.GetIncomingRequest().Reason.Record()
		if b, ok := c.builders[origin]; ok {
			b.act = act
			c.compose(b)
		} else {
			c.builders[origin] = &memberBuilder{act: act}
		}
	} else {
		c.requests[direct] = rec
	}
}

func (c *Composer) accountActivate(rec *record.Material) {
	log.Infof("account activate")
	direct := *rec.Virtual.GetActivate().Request.Record()
	if req, ok := c.requests[direct]; ok {
		origin := *req.Virtual.GetIncomingRequest().Reason.Record()
		if origin.Equal(insolar.ID{}) {
			delete(c.requests, origin)
			return
		}

		if b, ok := c.builders[origin]; ok {
			b.act = rec
			c.compose(b)
		} else {
			c.builders[origin] = &memberBuilder{act: rec}
		}
	} else {
		c.activates[direct] = rec
	}
}

func (c *Composer) compose(b *memberBuilder) {
	c.Lock()
	defer c.Unlock()

	member, err := b.build()
	if err == nil {
		c.cache = append(c.cache, member)
	}

	direct := *b.act.Virtual.GetActivate().Request.Record()
	origin := *b.res.Virtual.GetResult().Request.Record()
	delete(c.activates, direct)
	delete(c.requests, direct)
	delete(c.builders, origin)
}

func (c *Composer) Dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	log.Infof("dump members")
	c.updateStat()

	for _, member := range c.cache {
		if err := member.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump members")
		}
	}
	pub.Subscribe(func() {
		c.Lock()
		defer c.Unlock()
		c.cache = []*beauty.Member{}
	})
	return nil
}

func isMemberCreateRequest(req *record.Material) bool {
	request := (*dto.Request)(req)
	log.Infof("args %v", request.Virtual.GetIncomingRequest())
	if !request.IsMemberCall() {
		return false
	}

	log.Infof("member call")

	args := request.ParseMemberCallArguments()
	switch args.Params.CallSite {
	case "member.create", "member.migrationCreate":
		return true
	}
	return false
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
