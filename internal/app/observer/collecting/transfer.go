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
	"context"

	"github.com/insolar/insolar/application/builtin/contract/member"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

const (
	MemberCall = "Call"
)

type TransferCollector struct {
	log     *logrus.Logger
	fetcher store.RecordFetcher
}

func NewTransferCollector(log *logrus.Logger, fetcher store.RecordFetcher) *TransferCollector {
	c := &TransferCollector{
		log:     log,
		fetcher: fetcher,
	}
	return c
}

func (c *TransferCollector) Collect(ctx context.Context, rec *observer.Record) *observer.ExtendedTransfer {
	logger := c.log.WithField("collector", "transfer")

	result := observer.CastToResult(rec)
	if result == nil {
		return nil
	}

	if !result.IsSuccess() {
		return nil
	}

	requestID := result.Request()
	if requestID.IsEmpty() {
		logger.Error("failed to extract requestID from result")
		return nil
	}

	request, err := c.fetcher.Request(ctx, requestID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to fetch request"))
		return nil
	}

	incoming := request.Virtual.GetIncomingRequest()
	if incoming == nil {
		logger.Error("not a incoming request reason")
		return nil
	}

	if incoming.Method != MemberCall {
		logger.Debug("not a member call")
		return nil
	}

	transfer, err := c.build((*observer.Request)(&request), result)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build transfer"))
		return nil
	}
	return transfer
}

func (c *TransferCollector) build(request *observer.Request, result *observer.Result) (*observer.ExtendedTransfer, error) {
	callArguments := request.ParseMemberCallArguments()
	pn := request.ID.Pulse()
	callParams := &TransferCallParams{}
	request.ParseMemberContractCallParams(callParams)
	resultValue := &member.TransferResponse{Fee: "0"}
	result.ParseFirstPayloadValue(resultValue)
	memberFrom, err := insolar.NewIDFromString(callArguments.Params.Reference)
	if err != nil {
		return nil, errors.New("invalid fromMemberReference")
	}
	memberTo := memberFrom
	if callArguments.Params.CallSite == TransferMethod {
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

type TransferCallParams struct {
	Amount            string `json:"amount"`
	ToMemberReference string `json:"toMemberReference"`
	EthTxHash         string `json:"ethTxHash"`
}
