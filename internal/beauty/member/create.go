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
	"encoding/json"
	"sync"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member/signer"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/replication"
	log "github.com/sirupsen/logrus"
)

type memberBuilder struct {
	act *record.Material
	res *record.Material
}

func (b *memberBuilder) build() (*beauty.Member, error) {
	params := memberStatus(b.res.Virtual.GetResult().Payload)
	balance := initialBalance(b.act.Virtual.GetActivate())
	ref, err := insolar.NewReferenceFromBase58(params.reference)
	if err != nil {
		return nil, errors.New("invalid member reference")
	}
	return &beauty.Member{
		MemberRef:        ref.Record().String(),
		Balance:          balance,
		MigrationAddress: params.migrationAddress,
		WalletState:      b.act.ID.String(),
		Status:           params.status,
	}, nil
}

func NewComposer() *Composer {
	return &Composer{
		builders:  make(map[insolar.ID]*memberBuilder),
		requests:  make(map[insolar.ID]*record.Material),
		activates: make(map[insolar.ID]*record.Material),
		results:   make(map[insolar.ID]*record.Material),
	}
}

type Composer struct {
	builders  map[insolar.ID]*memberBuilder
	requests  map[insolar.ID]*record.Material
	activates map[insolar.ID]*record.Material
	results   map[insolar.ID]*record.Material

	sync.RWMutex
	cache []*beauty.Member
}

func (c *Composer) Process(rec *record.Material) {
	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Result:
		origin := *v.Result.Request.Record()
		if req, ok := c.requests[origin]; ok {
			delete(c.requests, origin)
			switch {
			case isMemberCreateRequest(req):
				c.memberCreateResult(rec)
			case isNewWallet(req):
				c.newWallet(req)
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
			case isNewWallet(rec):
				c.newWallet(rec)
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
		if isWalletActivate(v.Activate) {
			c.walletActivate(rec)
		}
	}
}

func (c *Composer) memberCreateResult(rec *record.Material) {
	origin := *rec.Virtual.GetResult().Request.Record()
	if b, ok := c.builders[origin]; ok {
		b.res = rec
		c.compose(b)
	} else {
		c.builders[origin] = &memberBuilder{res: rec}
	}
}

func (c *Composer) newWallet(rec *record.Material) {
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

func (c *Composer) walletActivate(rec *record.Material) {
	direct := *rec.Virtual.GetActivate().Request.Record()
	if req, ok := c.requests[direct]; ok {
		origin := *req.Virtual.GetIncomingRequest().Reason.Record()
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
	log.Infof("ready %d", len(c.cache))
	log.Infof("req %d", len(c.requests))
	log.Infof("res %d", len(c.results))
	log.Infof("act %d", len(c.activates))
	log.Infof("builders %d", len(c.builders))
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
	_, ok := req.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		return false
	}
	in := req.Virtual.GetIncomingRequest()
	if in.Method != "Call" {
		return false
	}

	args := parseCallArguments(in.Arguments)
	switch args.Params.CallSite {
	case "member.create", "member.migrationCreate":
		return true
	}
	return false
}

func parseCallArguments(inArgs []byte) member.Request {
	var args []interface{}
	err := insolar.Deserialize(inArgs, &args)
	if err != nil {
		log.Warn(errors.Wrapf(err, "failed to deserialize request arguments"))
		return member.Request{}
	}

	request := member.Request{}
	if len(args) > 0 {
		if rawRequest, ok := args[0].([]byte); ok {
			var (
				pulseTimeStamp int64
				signature      string
				raw            []byte
			)
			err = signer.UnmarshalParams(rawRequest, &raw, &signature, &pulseTimeStamp)
			if err != nil {
				log.Warn(errors.Wrapf(err, "failed to unmarshal params"))
				return member.Request{}
			}
			err = json.Unmarshal(raw, &request)
			if err != nil {
				log.Warn(errors.Wrapf(err, "failed to unmarshal json member request"))
				return member.Request{}
			}
		}
	}
	return request
}
