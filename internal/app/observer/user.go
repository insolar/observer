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

type User struct {
	UserRef   insolar.Reference
	KYCStatus bool
	Public    string
	Status    string
	State     []byte
}

type UserKYC struct {
	PrevState insolar.ID
	UserState insolar.ID
	KYC       bool
	Source    string
	Timestamp int64
}

type UserStorage interface {
	Insert(User) error
	Update(UserKYC) error
}

type UserCollector interface {
	Collect(*Record) *User
}
