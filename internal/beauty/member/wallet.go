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
	"github.com/insolar/insolar/insolar/record"
	proxyWallet "github.com/insolar/insolar/logicrunner/builtin/proxy/wallet"
)

func isWalletActivate(act *record.Activate) bool {
	return act.Image.Equal(*proxyWallet.PrototypeReference)
}

func isNewWallet(rec *record.Material) bool {
	_, ok := rec.Virtual.Union.(*record.Virtual_IncomingRequest)
	if !ok {
		return false
	}
	in := rec.Virtual.GetIncomingRequest()
	if in.Method != "New" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	return in.Prototype.Equal(*proxyWallet.PrototypeReference)
}

func isWalletAmend(amd *record.Amend) bool {
	return amd.Image.Equal(*proxyWallet.PrototypeReference)
}
