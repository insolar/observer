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

package transfer

import (
	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/pulse"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/dto"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replication"
)

func build(req *record.Material, res *record.Material) (*beauty.Transfer, error) {
	// TODO: add wallet refs
	callArguments := (*dto.Request)(req).ParseMemberCallArguments()
	pn := req.ID.Pulse()
	callParams := parseTransferCallParams(callArguments)
	transferResult := parseTransferResultPayload(res)
	memberFrom, err := insolar.NewReferenceFromBase58(callArguments.Params.Reference)
	if err != nil {
		return nil, errors.New("invalid fromMemberReference")
	}
	memberTo, err := insolar.NewReferenceFromBase58(callParams.toMemberReference)
	if err != nil {
		return nil, errors.New("invalid toMemberReference")
	}
	transferDate, err := pulse.Number(pn).AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert transfer pulse to time")
	}
	return &beauty.Transfer{
		TxID:          insolar.NewReference(req.ID).String(),
		Status:        string(transferResult.status),
		Amount:        callParams.amount,
		MemberFromRef: memberFrom.String(),
		MemberToRef:   memberTo.String(),
		PulseNum:      pn,
		TransferDate:  transferDate.Unix(),
		Fee:           transferResult.fee,
		WalletFromRef: "TODO",
		WalletToRef:   "TODO",
		EthHash:       "",
	}, nil
}

type Composer struct {
	requests map[insolar.ID]*record.Material
	results  map[insolar.ID]*record.Material
	cache    []*beauty.Transfer

	stat *dumpStat
}

func NewComposer() *Composer {
	stat := &dumpStat{
		cached: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "observer_member_transfer_composer_cached_total",
			Help: "Cache size of migration address composer",
		}),
	}
	return &Composer{
		requests: make(map[insolar.ID]*record.Material),
		results:  make(map[insolar.ID]*record.Material),
		stat:     stat,
	}
}

func (c *Composer) Process(rec *record.Material) {
	defer panic.Log("member_transfer_composer")

	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Result:
		origin := *v.Result.Request.Record()
		if req, ok := c.requests[origin]; ok {
			delete(c.requests, origin)
			request := (*dto.Request)(req)
			if request.IsIncoming() {
				if isTransferCall(request) {
					if transfer, err := build(req, rec); err == nil {
						c.cache = append(c.cache, transfer)
					}
				}
			}
		} else {
			c.results[origin] = rec
		}
	case *record.Virtual_IncomingRequest:
		origin := rec.ID
		if res, ok := c.results[origin]; ok {
			delete(c.results, origin)
			request := (*dto.Request)(rec)
			if isTransferCall(request) {
				if transfer, err := build(rec, res); err == nil {
					c.cache = append(c.cache, transfer)
				}
			}
		} else {
			c.requests[origin] = rec
		}
	case *record.Virtual_OutgoingRequest:
		origin := rec.ID
		if _, ok := c.results[origin]; ok {
			delete(c.results, origin)
		} else {
			c.requests[origin] = rec
		}
	}
}

func isTransferCall(request *dto.Request) bool {
	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "member.transfer"
}

func (c *Composer) Dump(tx *pg.Tx, pub replication.OnDumpSuccess) error {
	log.Infof("dump member transfers")

	c.updateStat()

	for _, transfer := range c.cache {
		if err := transfer.Dump(tx); err != nil {
			return errors.Wrapf(err, "failed to dump transfers")
		}
	}

	pub.Subscribe(func() {
		c.cache = []*beauty.Transfer{}
	})
	return nil
}

type dumpStat struct {
	cached prometheus.Gauge
}

func (c *Composer) updateStat() {
	requestCount := len(c.requests)
	resultCount := len(c.results)

	c.stat.cached.Set(float64(requestCount + resultCount))
}
