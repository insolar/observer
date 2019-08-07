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

package deposit

// func (b *beauty.Beautifier) processDepositAmend(id insolar.ID, amd *record.Amend) {
// 	deposit := depositState(amd)
// 	b.depositUpdates[id] = beauty.DepositUpdate{
// 		id:        id,
// 		amount:    deposit.Amount,
// 		withdrawn: "0",
// 		status:    "MIGRATION",
// 		prevState: amd.PrevState.String(),
// 	}
// }
//
// func depositState(amd *record.Amend) *deposit.Deposit {
// 	d := deposit.Deposit{}
// 	err := insolar.Deserialize(amd.Memory, &d)
// 	if err != nil {
// 		log.Error(errors.New("failed to deserialize deposit contract state"))
// 	}
// 	return &d
// }
