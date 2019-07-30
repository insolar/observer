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

package beauty

type Deposit struct {
	tableName struct{} `sql:"deposits"`

	ID              uint `sql:",pk_id"`
	Timestamp       uint
	HoldReleaseDate uint
	Amount          string
	Bonus           string
	EthHash         string
	Status          string
	MemberID        uint
}

func (b *Beautifier) storeDeposit(deposit *Deposit) error {
	_, err := b.db.Model(deposit).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
