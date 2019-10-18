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
	"encoding/base64"

	"github.com/insolar/insolar/application"
	"github.com/insolar/insolar/insolar"

	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	proxyMigrationAdmin "github.com/insolar/insolar/application/builtin/proxy/migrationadmin"
	proxyDaemon "github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	"github.com/insolar/insolar/application/genesisrefs"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/panic"
)

type ToDepositTransferCollector struct {
	log             *logrus.Logger
	memberAddresses observer.ResultCollector
	transferResults observer.ResultCollector
	confirmChains   observer.ChainCollector
	halfChains      observer.ChainCollector
	chains          observer.ChainCollector
}

func NewToDepositTransferCollector(log *logrus.Logger) *ToDepositTransferCollector {
	c := &ToDepositTransferCollector{
		log: log,
	}
	c.memberAddresses = NewResultCollector(c.isGetMemberByMigrationAddress, c.successResult)
	c.transferResults = NewResultCollector(c.isTransferToDeposit, c.successResult)
	c.confirmChains = NewChainCollector(&RelationDesc{
		Is: c.isConfirmDeposit,
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: c.isConfirmDeposit,
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return c.isTransferToDeposit(couple.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return insolar.ID{}
			}
			return couple.Request.Reason()
		},
		Proper: func(chain interface{}) bool {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return c.isTransferToDeposit(couple.Request)
		},
	})

	c.halfChains = NewChainCollector(&RelationDesc{
		Is: c.isDepositMigrationCall,
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: c.isDepositMigrationCall,
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return c.isGetMemberByMigrationAddress(couple.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return insolar.ID{}
			}
			return couple.Request.Reason()
		},
		Proper: func(chain interface{}) bool {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return c.isGetMemberByMigrationAddress(couple.Request)
		},
	})

	c.chains = NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isDepositMigrationCall(ch.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(ch.Parent)
			return request.ID
		},
		Proper: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isDepositMigrationCall(ch.Parent)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isConfirmDeposit(ch.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(ch.Parent)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isConfirmDeposit(ch.Parent)
		},
	})
	return c
}

func (c *ToDepositTransferCollector) Collect(rec *observer.Record) *observer.ExtendedTransfer {
	defer panic.Catch("deposit_confirm_transfer_collector")

	if rec == nil {
		return nil
	}

	transferCouple := c.transferResults.Collect(rec)
	confirmChain := c.confirmChains.Collect(rec)
	if transferCouple != nil {
		confirmChain = c.confirmChains.Collect(transferCouple)
	}

	addressCouple := c.memberAddresses.Collect(rec)
	half := c.halfChains.Collect(rec)
	if addressCouple != nil {
		half = c.halfChains.Collect(addressCouple)
	}

	var chain *observer.Chain
	if confirmChain != nil {
		chain = c.chains.Collect(confirmChain)
	}
	if half != nil {
		chain = c.chains.Collect(half)
	}

	if chain == nil {
		return nil
	}

	addressRes, confirm, transferReq := c.unwrapChain(chain)
	transfer, err := c.build(addressRes, confirm, transferReq)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build transfer"))
		return nil
	}
	return transfer
}

func (c *ToDepositTransferCollector) unwrapChain(chain *observer.Chain) (*observer.Result, *observer.Request, *observer.Request) {
	half, ok := chain.Parent.(*observer.Chain)
	if !ok {
		c.log.Errorf("trying to use %T as *observer.Chain", chain.Parent)
		return nil, nil, nil
	}

	addressCouple, ok := half.Child.(*observer.CoupledResult)
	if !ok {
		c.log.Errorf("trying to use %T as *observer.CoupledResult", half.Child)
		return nil, nil, nil
	}

	addressRes := addressCouple.Result

	comfirmChain, ok := chain.Child.(*observer.Chain)
	if !ok {
		c.log.Errorf("trying to use %T as *observer.Chain", chain.Child)
		return nil, nil, nil
	}

	couple, ok := comfirmChain.Child.(*observer.CoupledResult)
	if !ok {
		c.log.Errorf("trying to use %T as *observer.CoupledResult", comfirmChain.Child)
		return nil, nil, nil

	}
	transfer := couple.Request

	req, ok := comfirmChain.Parent.(*observer.Record)
	if !ok {
		c.log.Errorf("trying to use %T as *observer.Record", comfirmChain.Parent)
	}

	confirm := observer.CastToRequest(req)
	return addressRes, confirm, transfer
}

func (c *ToDepositTransferCollector) build(
	addressRes *observer.Result,
	confirm *observer.Request,
	transfer *observer.Request,
) (*observer.ExtendedTransfer, error) {
	var (
		daemonRef string
		txHash    string
		amount    string
	)

	memberFrom := genesisrefs.GenesisRef(application.GenesisNameMigrationAdminMember)
	var refTo string
	addressRes.ParseFirstPayloadValue(&refTo)
	buf, err := base64.StdEncoding.DecodeString(refTo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deserialize memberTo reference from base64")
	}
	memberTo := insolar.NewReferenceFromBytes(buf)
	if memberTo == nil {
		return nil, errors.Wrapf(err, "failed to deserialize memberTo reference from result record")
	}
	confirm.ParseIncomingArguments(&daemonRef, &txHash, &amount)
	pn := transfer.ID.Pulse()
	transferDate, err := pn.AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert transfer pulse to time")
	}
	return &observer.ExtendedTransfer{
		DepositTransfer: observer.DepositTransfer{
			Transfer: observer.Transfer{
				TxID:      transfer.ID,
				From:      *memberFrom.GetLocal(),
				To:        *memberTo.GetLocal(),
				Amount:    amount,
				Fee:       "0",
				Timestamp: transferDate.Unix(),
				Pulse:     pn,
			},
			EthHash: txHash,
		},
	}, nil
}

func (c *ToDepositTransferCollector) isGetMemberByMigrationAddress(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "GetMemberByMigrationAddress" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*proxyMigrationAdmin.PrototypeReference)
}

func (c *ToDepositTransferCollector) isDepositMigrationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "DepositMigrationCall" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*proxyDaemon.PrototypeReference)
}

func (c *ToDepositTransferCollector) isConfirmDeposit(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "Confirm" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func (c *ToDepositTransferCollector) isTransferToDeposit(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "TransferToDeposit" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*proxyDeposit.PrototypeReference)
}

func (c *ToDepositTransferCollector) successResult(chain interface{}) bool {
	result := observer.CastToResult(chain)
	return result.IsSuccess()
}
