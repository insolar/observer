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

import (
	"context"
	"fmt"
	"github.com/insolar/observer/internal/beauty/models"
	"github.com/insolar/observer/internal/ledger/store"
	"github.com/jinzhu/gorm"
)

func NewBeautifier() *Beautifier {
	return &Beautifier{}
}

type Beautifier struct {
	DB store.DB `inject:""`
}

func (b *Beautifier) Init(ctx context.Context) error {
	//inslogger.FromContext(context.Background()).Infof("Initialize db connections %v", b.DB)
	db, err := gorm.Open("postgres", "host=localhost port=5432 user=hirama dbname=postgres sslmode=disable")
	if err != nil {
		fmt.Println("AAAAAA", err.Error())
	}
	db.AutoMigrate(models.InsTransaction{})
	var tx = models.InsTransaction{TxID: "foo", Amount: "100000", Fee: "1000", TimeStamp: 123123, Pulse: 12300,
		Status: "SUCCESS", ReferenceFrom: "bar", ReferenceTo: "tar"}
	db.Save(&tx)
	defer db.Close()
	return nil
}

func (b *Beautifier) Start(ctx context.Context) error {
	// WorkFlow
	// Start from previous work
	// Take chunk of raw data and insert in db (it can be tx or account creation)
	// save done work in db
	return nil
}

func (b *Beautifier) Stop(ctx context.Context) error {
	return nil
}
