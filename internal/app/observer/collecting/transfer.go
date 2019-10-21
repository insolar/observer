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

package collecting

import (
	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
)

type TransferCollector struct {
	log       *logrus.Logger
	collector observer.ResultCollector
}

func NewTransferCollector(log *logrus.Logger) *TransferCollector {
	c := &TransferCollector{
		log: log,
	}
	c.collector = NewResultCollector(c.isTransferCall, c.successResult)
	return c
}

func (c *TransferCollector) Collect(rec *observer.Record) *observer.ExtendedTransfer {
	couple := c.collector.Collect(rec)
	if couple == nil {
		return nil
	}
	transfer, err := c.build(couple.Request, couple.Result)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build transfer"))
		return nil
	}
	return transfer
}

func (c *TransferCollector) isTransferCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)

	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "deposit.transfer"
}

func (c *TransferCollector) successResult(chain interface{}) bool {
	result := observer.CastToResult(chain)
	return result.IsSuccess()
}

func (c *TransferCollector) build(request *observer.Request, result *observer.Result) (*observer.ExtendedTransfer, error) {
	callArguments := request.ParseMemberCallArguments()
	pn := request.ID.Pulse()
	callParams := &transferCallParams{}
	request.ParseMemberContractCallParams(callParams)
	resultValue := &member.TransferResponse{Fee: "0"}
	result.ParseFirstPayloadValue(resultValue)
	memberFrom, err := insolar.NewIDFromString(callArguments.Params.Reference)
	if err != nil {
		return nil, errors.New("invalid fromMemberReference")
	}
	memberTo := memberFrom
	if callArguments.Params.CallSite == transferMethod {
		memberTo, err = insolar.NewIDFromString(callParams.ToMemberReference)
		if err != nil {
			return nil, errors.New("invalid toMemberReference")
		}
	}

	transferDate, err := pn.AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert transfer pulse to time")
	}
	return &observer.ExtendedTransfer{
		DepositTransfer: observer.DepositTransfer{
			Transfer: observer.Transfer{
				TxID:      request.ID,
				Amount:    callParams.Amount,
				From:      *memberFrom,
				To:        *memberTo,
				Pulse:     pn,
				Timestamp: transferDate.Unix(),
				Fee:       resultValue.Fee,
			},
			EthHash: callParams.EthTxHash,
		},
	}, nil
}

type transferCallParams struct {
	Amount            string `json:"amount"`
	ToMemberReference string `json:"toMemberReference"`
	EthTxHash         string `json:"ethTxHash"`
}
