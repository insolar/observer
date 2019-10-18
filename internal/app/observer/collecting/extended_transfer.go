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
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	proxyAccount "github.com/insolar/insolar/logicrunner/builtin/proxy/account"
	proxyCostCenter "github.com/insolar/insolar/logicrunner/builtin/proxy/costcenter"
	proxyMember "github.com/insolar/insolar/logicrunner/builtin/proxy/member"
	proxyWallet "github.com/insolar/insolar/logicrunner/builtin/proxy/wallet"

	"github.com/insolar/observer/internal/app/observer"
)

type ExtendedTransferCollector struct {
	log *logrus.Logger

	// 1st level
	rootResults      observer.ResultCollector
	feeMemberResults observer.ResultCollector
	calcFeeRequests  observer.ChainCollector

	// 2nd
	feeChains observer.ChainCollector

	// 3rd
	accountTransferChains observer.ChainCollector

	// 4th
	walletTransferChains observer.ChainCollector

	// final
	chains observer.ChainCollector
}

func NewExtendedTransferCollector(log *logrus.Logger) *ExtendedTransferCollector {
	c := &ExtendedTransferCollector{
		log: log,
	}

	// 1st
	rootResults := NewResultCollector(c.isTransferCall, successResult)
	feeMemberResults := NewResultCollector(c.isGetFeeMember, successResult)

	calcFeeRequests := NewChainCollector(&RelationDesc{
		Is: c.isAccountTransfer,
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: c.isAccountTransfer,
	}, &RelationDesc{
		Is: c.isCalcFee,
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.Reason()
		},
		Proper: c.isCalcFee,
	})

	// 2nd
	feeChain := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isAccountTransfer(ch.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(c.Parent)
			return request.ID
		},
		Proper: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isAccountTransfer(ch.Parent)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return c.isGetFeeMember(couple.Request)
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
			return c.isGetFeeMember(couple.Request)
		},
	})

	// 3rd
	accountTransferChains := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			ch, ok = ch.Parent.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isAccountTransfer(ch.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			c, ok = c.Parent.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(c.Parent)
			return request.ID
		},
		Proper: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			ch, ok = ch.Parent.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isAccountTransfer(ch.Parent)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			_, ok := chain.(*observer.Chain)
			if ok {
				return false
			}
			request := observer.CastToRequest(chain)
			return request.IsIncoming()
		},
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			_, ok := chain.(*observer.Chain)
			if ok {
				return false
			}
			return c.isMemberAccept(chain)
		},
	})

	// 4th
	walletTransferChains := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			request := observer.CastToRequest(chain)
			return request.IsIncoming() || request.IsOutgoing()
		},
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.ID
		},
		Proper: c.isWalletTransfer,
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			ch, ok = ch.Parent.(*observer.Chain)
			if !ok {
				return false
			}
			ch, ok = ch.Parent.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isAccountTransfer(ch.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			c, ok = c.Parent.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			c, ok = c.Parent.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(c.Parent)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			ch, ok = ch.Parent.(*observer.Chain)
			if !ok {
				return false
			}
			ch, ok = ch.Parent.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isAccountTransfer(ch.Parent)
		},
	})

	// final
	chains := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return c.isTransferCall(couple.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return insolar.ID{}
			}
			return couple.Request.ID
		},
		Proper: func(chain interface{}) bool {
			couple, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return c.isTransferCall(couple.Request)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isWalletTransfer(ch.Parent)
		},
		Origin: func(chain interface{}) insolar.ID {
			c, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			request := observer.CastToRequest(c.Parent)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			ch, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			return c.isWalletTransfer(ch.Parent)
		},
	})

	// 1st
	c.rootResults = rootResults
	c.feeMemberResults = feeMemberResults
	c.calcFeeRequests = calcFeeRequests

	// 2nd
	c.feeChains = feeChain

	// 3rd
	c.accountTransferChains = accountTransferChains

	// 4th
	c.walletTransferChains = walletTransferChains

	// final
	c.chains = chains

	return c
}

func (c *ExtendedTransferCollector) Collect(rec *observer.Record) *observer.ExtendedTransfer {
	if rec == nil {
		return nil
	}

	// 1st
	rootResult := c.rootResults.Collect(rec)
	feeMemberResult := c.feeMemberResults.Collect(rec)
	calcFeeRequest := c.calcFeeRequests.Collect(rec)

	// 2nd
	var feeChain *observer.Chain
	if calcFeeRequest != nil {
		feeChain = c.feeChains.Collect(calcFeeRequest)
	}
	if feeMemberResult != nil {
		feeChain = c.feeChains.Collect(feeMemberResult)
	}

	// 3rd
	var accountTransfer *observer.Chain
	if feeChain != nil {
		accountTransfer = c.accountTransferChains.Collect(feeChain)
	}
	if c.isMemberAccept(rec) {
		accountTransfer = c.accountTransferChains.Collect(rec)
	}

	// 4th
	walletTransfer := c.walletTransferChains.Collect(rec)
	if accountTransfer != nil {
		walletTransfer = c.walletTransferChains.Collect(accountTransfer)
	}

	// final
	var chain *observer.Chain
	if walletTransfer != nil {
		chain = c.chains.Collect(walletTransfer)
	}
	if rootResult != nil {
		chain = c.chains.Collect(rootResult)
	}

	if chain == nil {
		return nil
	}
	root, result, wallet, account, calc, getFeeMember, feeMember, acpt := c.unwrapChain(chain)
	transfer, err := c.build(root, result, wallet, account, calc, getFeeMember, feeMember, acpt)
	if err != nil {
		c.log.Error(errors.Wrapf(err, "failed to build transfer"))
		return nil
	}
	return transfer
}

func (c *ExtendedTransferCollector) unwrapChain(chain *observer.Chain) (
	*observer.Request,
	*observer.Result,
	*observer.Request,
	*observer.Request,
	*observer.Request,
	*observer.Request,
	*observer.Result,
	*observer.Request,
) {
	rootResult, ok := chain.Parent.(*observer.CoupledResult)
	if !ok {
		c.log.Errorf("[0] trying to use %T as *observer.CoupledResult", chain.Parent)
		return nil, nil, nil, nil, nil, nil, nil, nil
	}

	root := rootResult.Request
	result := rootResult.Result

	walletTransfer, ok := chain.Child.(*observer.Chain)
	if !ok {
		c.log.Errorf("[1] trying to use %T as &observer.Chain", chain.Child)
	}
	wallet := observer.CastToRequest(walletTransfer.Parent)

	accoutTransfer, ok := walletTransfer.Child.(*observer.Chain)
	if !ok {
		c.log.Errorf("[2] trying to use %T as &observer.Chain", walletTransfer.Child)
	}
	accept := observer.CastToRequest(accoutTransfer.Child)

	feeChain, ok := accoutTransfer.Parent.(*observer.Chain)
	if !ok {
		c.log.Errorf("[3] trying to use %T as &observer.Chain", accoutTransfer.Parent)
	}

	calcFeeRequestPair, ok := feeChain.Parent.(*observer.Chain)
	if !ok {
		c.log.Errorf("[4] trying to use %T as *observer.Chain", chain.Parent)
		return nil, nil, nil, nil, nil, nil, nil, nil
	}

	account := observer.CastToRequest(calcFeeRequestPair.Parent)
	calc := observer.CastToRequest(calcFeeRequestPair.Child)
	getFeeMemberPair, ok := feeChain.Child.(*observer.CoupledResult)
	if !ok {
		c.log.Errorf("[5] trying to use %T as *observer.CoupledResult", chain.Parent)
		return nil, nil, nil, nil, nil, nil, nil, nil
	}
	getFeeMember := getFeeMemberPair.Request
	feeMember := getFeeMemberPair.Result
	return root, result, wallet, account, calc, getFeeMember, feeMember, accept
}

func (c *ExtendedTransferCollector) build(
	root *observer.Request,
	result *observer.Result,
	wallet *observer.Request,
	account *observer.Request,
	calc *observer.Request,
	getFeeMember *observer.Request,
	feeMemberResult *observer.Result,
	accept *observer.Request,
) (*observer.ExtendedTransfer, error) {

	simpleCollector := &TransferCollector{log: c.log}
	simpleTransfer, err := simpleCollector.build(root, result)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build simple transfer")
	}
	costCenterRef := calc.Virtual.GetIncomingRequest().Object
	costCenter := insolar.ID{}
	if costCenterRef != nil {
		costCenter = *costCenterRef.GetLocal()
	}

	feeMember := insolar.ID{}
	if feeMemberResult.IsResult() {
		ret, _ := feeMemberResult.ParsePayload()
		buf, ok := ret.Returns[0].([]byte)
		if !ok {
			return nil, errors.Wrapf(err, "failed to cast GetFeeMember return as byte slice")
		}
		feeMemberRef := insolar.NewReferenceFromBytes(buf)
		feeMember = *feeMemberRef.GetLocal()
	}
	return &observer.ExtendedTransfer{
		DepositTransfer:        simpleTransfer.DepositTransfer,
		TransferRequestMember:  root.ID,
		TransferRequestWallet:  wallet.ID,
		TransferRequestAccount: account.ID,
		AcceptRequestMember:    accept.ID,
		CalcFeeRequest:         calc.ID,
		FeeMemberRequest:       getFeeMember.ID,
		CostCenterRef:          costCenter,
		FeeMemberRef:           feeMember,
	}, nil
}

func (c *ExtendedTransferCollector) isTransferCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)

	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}

	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "member.transfer"
}

func (c *ExtendedTransferCollector) isGetFeeMember(chain interface{}) bool {
	request := observer.CastToRequest(chain)

	if !request.IsIncoming() {
		return false
	}

	req := request.Virtual.GetIncomingRequest()
	if req.Method != "GetFeeMember" {
		return false
	}
	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyCostCenter.PrototypeReference)
}

func (c *ExtendedTransferCollector) isAccountTransfer(chain interface{}) bool {
	request := observer.CastToRequest(chain)

	if !request.IsIncoming() {
		return false
	}

	req := request.Virtual.GetIncomingRequest()
	if req.Method != "Transfer" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyAccount.PrototypeReference)
}

func (c *ExtendedTransferCollector) isMemberAccept(chain interface{}) bool {
	request := observer.CastToRequest(chain)

	if !request.IsIncoming() {
		return false
	}

	req := request.Virtual.GetIncomingRequest()
	if req.Method != "Accept" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyMember.PrototypeReference)
}

func (c *ExtendedTransferCollector) isCalcFee(chain interface{}) bool {
	request := observer.CastToRequest(chain)

	if !request.IsIncoming() {
		return false
	}

	req := request.Virtual.GetIncomingRequest()
	if req.Method != "CalcFee" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyCostCenter.PrototypeReference)
}

func (c *ExtendedTransferCollector) isWalletTransfer(chain interface{}) bool {
	request := observer.CastToRequest(chain)

	if !request.IsIncoming() {
		return false
	}

	req := request.Virtual.GetIncomingRequest()
	if req.Method != "Transfer" {
		return false
	}

	if req.Prototype == nil {
		return false
	}
	return req.Prototype.Equal(*proxyWallet.PrototypeReference)
}
