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
	"context"
	"encoding/hex"

	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/pulse"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/dto"
	"github.com/insolar/observer/internal/metrics"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/panic"
	"github.com/insolar/observer/internal/replicator"
)

type transferCallParams struct {
	Amount            string `json:"amount"`
	ToMemberReference string `json:"toMemberReference"`
	EthTxHash         string `json:"ethTxHash"`
}

type transferResult struct {
	Fee string `json:"fee"`
}

func build(req *record.Material, res *record.Material) (*beauty.Transfer, error) {
	// TODO: add wallet refs
	request := (*dto.Request)(req)
	result := (*dto.Result)(res)
	callArguments := request.ParseMemberCallArguments()
	pn := req.ID.Pulse()
	callParams := &transferCallParams{}
	request.ParseMemberContractCallParams(callParams)
	status := dto.SUCCESS
	if !result.IsSuccess() {
		status = dto.CANCELED
	}
	resultValue := &transferResult{Fee: "0"}
	result.ParseFirstPayloadValue(resultValue)
	memberFrom, err := insolar.NewReferenceFromBase58(callArguments.Params.Reference)
	if err != nil {
		return nil, errors.New("invalid fromMemberReference")
	}
	to := ""
	switch callArguments.Params.CallSite {
	case "member.transfer":
		memberTo, err := insolar.NewReferenceFromBase58(callParams.ToMemberReference)
		if err != nil {
			return nil, errors.New("invalid toMemberReference")
		}
		to = memberTo.String()
	case "deposit.transfer":
		to = memberFrom.String()
	}

	transferDate, err := pulse.Number(pn).AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert transfer pulse to time")
	}
	return &beauty.Transfer{
		TxID:          insolar.NewReference(req.ID).String(),
		Status:        string(status),
		Amount:        callParams.Amount,
		MemberFromRef: memberFrom.String(),
		MemberToRef:   to,
		PulseNum:      pn,
		TransferDate:  transferDate.Unix(),
		Fee:           resultValue.Fee,
		WalletFromRef: "TODO",
		WalletToRef:   "TODO",
		EthHash:       callParams.EthTxHash,
	}, nil
}

type Composer struct {
	Metrics metrics.Registry `inject:""`

	requests map[insolar.ID]*record.Material
	results  map[insolar.ID]*record.Material
	cache    []*beauty.Transfer

	stat *dumpStat
}

func NewComposer() *Composer {
	stat := &dumpStat{
		cached: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "observer_transfer_composer_cached_total",
			Help: "Cache size of migration address composer",
		}),
	}
	return &Composer{
		requests: make(map[insolar.ID]*record.Material),
		results:  make(map[insolar.ID]*record.Material),
		stat:     stat,
	}
}

func (c *Composer) Init(ctx context.Context) error {
	if c.Metrics != nil {
		c.Metrics.Register(c.stat.cached)
	}
	return nil
}

func (c *Composer) Process(rec *record.Material) {
	defer panic.Log("member_transfer_composer")

	switch v := rec.Virtual.Union.(type) {
	case *record.Virtual_Result:
		origin := *v.Result.Request.GetLocal()
		if req, ok := c.requests[origin]; ok {
			delete(c.requests, origin)
			request := (*dto.Request)(req)
			if request.IsIncoming() {
				if isTransferCall(request) {
					transferCall, _ := req.Marshal()
					log.Infof("transfer call: %s", hex.EncodeToString(transferCall))
					transferResult, _ := rec.Marshal()
					log.Infof("transfer result: %s", hex.EncodeToString(transferResult))
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
				transferCall, _ := rec.Marshal()
				log.Infof("transfer call: %s", hex.EncodeToString(transferCall))
				transferResult, _ := res.Marshal()
				log.Infof("transfer result: %s", hex.EncodeToString(transferResult))
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
	switch args.Params.CallSite {
	case "member.transfer", "deposit.transfer":
		return true
	}
	return false
}

func (c *Composer) Dump(tx orm.DB, pub replicator.OnDumpSuccess) error {
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