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

import (
	"context"

	"github.com/insolar/insolar/insolar"
)

// Member describes insolar member.
type Member struct {
	MemberRef        insolar.Reference
	Balance          string
	MigrationAddress string
	AccountState     insolar.ID
	Status           string
	WalletRef        insolar.Reference
	AccountRef       insolar.Reference
	PublicKey        string
}

type Balance struct {
	PrevState    insolar.ID
	AccountState insolar.ID
	Balance      string
}

type MemberCollector interface {
	Collect(context.Context, *Record) *Member
}

type BalanceCollector interface {
	Collect(*Record) *Balance
}

type MemberStorage interface {
	Insert(*Member) error
	Update(*Balance) error
}

type BalanceFilter interface {
	Filter(map[insolar.ID]*Balance, map[insolar.ID]*Member)
}
