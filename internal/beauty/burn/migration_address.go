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

package burn

import (
	"encoding/json"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	"github.com/insolar/insolar/logicrunner/builtin/contract/member/signer"
	"github.com/insolar/insolar/network/consensus/common/pulse"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/replication"
)

type Composer struct {
	cache []*beauty.MigrationAddress
}

func NewComposer() *Composer {
	return &Composer{}
}

func (c *Composer) Process(rec *record.Material) {
	if isAddBurnAddresses(rec) {
		c.processAddBurnAddresses(rec)
	}
}

func (c *Composer) Dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	for _, addr := range c.cache {
		if err := addr.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump migration addresses")
		}
	}

	pub.Subscribe(func() {
		c.cache = []*beauty.MigrationAddress{}
	})
	return nil
}

func (c *Composer) processAddBurnAddresses(rec *record.Material) {
	in := rec.Virtual.GetIncomingRequest()
	args := parseCallArguments(in.Arguments)
	addresses := parseAddBurnAddressesCallParams(args)
	pn := pulse.Number(rec.ID.Pulse())
	for _, addr := range addresses {
		c.cache = append(c.cache, &beauty.MigrationAddress{
			Addr:      addr,
			Timestamp: pn.AsApproximateTime().Unix(),
			Wasted:    false,
		})
	}
}

func isAddBurnAddresses(rec *record.Material) bool {
	v, ok := rec.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		return false
	}

	in := v.IncomingRequest
	if in.Method != "Call" {
		return false
	}

	args := parseCallArguments(in.Arguments)
	return args.Params.CallSite == "migration.addBurnAddresses"
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
