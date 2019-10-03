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

package observer

import "github.com/insolar/insolar/insolar"

// Transfer describes token moving between the insolar members.
type Transfer struct {
	TxID      insolar.ID
	From      insolar.ID
	To        insolar.ID
	Amount    string
	Fee       string
	Timestamp int64
	Pulse     insolar.PulseNumber
}

// DepositTransfer describes token moving from deposit account to the insolar member account.
type DepositTransfer struct {
	Transfer
	EthHash string
}

// <column name="transter_request_member" type="bytea"/>
// <column name="transter_request_wallet" type="bytea"/>
// <column name="transter_request_account" type="bytea"/>
// <column name="accept_request_member" type="bytea"/>
// <column name="accept_request_wallet" type="bytea"/>
// <column name="accept_request_account" type="bytea"/>
// <column name="calc_fee_request" type="bytea"/>
// <column name="fee_member_accept" type="bytea"/>
// <column name="costcenter_ref" type="bytea"/>
// <column name="fee_member_ref" type="bytea"/>
type ExtendedTransfer struct {
	DepositTransfer
	TransferRequestMember  insolar.ID // member.Call
	TransferRequestWallet  insolar.ID // wallet.Transfer or empty for transfer from deposit
	TransferRequestAccount insolar.ID // account.Transfer or account.TransferToDeposit
	AcceptRequestMember    insolar.ID // always empty
	AcceptRequestWallet    insolar.ID // always empty
	AcceptRequestAccount   insolar.ID // account.Accept or deposit.Accept
	CalcFeeRequest         insolar.ID // costcenter.CalcFee or empty
	FeeMemberRequest       insolar.ID // costcenter.GetFeeMember or empty
	CostCenterRef          insolar.ID // from arg costcenter.GetObject or empty
	FeeMemberRef           insolar.ID // result from cc.GetFeeMember or empty
}

type TransferStorage interface {
	Insert(*Transfer) error
}

type TransferCollector interface {
	Collect(*Record) *DepositTransfer
}
