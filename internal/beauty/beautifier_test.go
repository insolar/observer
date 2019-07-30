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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/go-pg/pg/orm"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/observer/internal/ledger/store"
)

type publisherStub struct{}

func (p *publisherStub) Subscribe(handle store.DBSetHandle) {}

func TestBeautifier_parse(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()
	b.Publisher = &publisherStub{}

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	key, _ := hex.DecodeString("01acf05e0acd805d7c8b19d544baef789db976f31cd9407bc69fc62ebab6ddf9")
	value, _ := hex.DecodeString("a20169b20666a201200000000000000000000000000000000000000000000000000000000000000000aa014001acf05eb0c4c83a7b7052955db36e6f26063259449b42e94f6f03e700d1ba580000000000000000000000000000000000000000000000000000000000000000aa01200000000103a00000000000000000000000000000000000000000000000000000")

	b.process(key, value, 1)
}

func TestBeautifier_storeTx(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()
	b.Publisher = &publisherStub{}

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	tx := Transaction{TxID: "foo", Amount: "1000", Fee: "100",
		Timestamp: 5555, Pulse: 32323, Status: "SUCCESS", ReferenceTo: "alpha", ReferenceFrom: "beta"}

	assert.NoError(t, b.storeTx(&tx))
}

func TestBeautifier_ParseAndStore(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()
	b.Publisher = &publisherStub{}

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	var records []Record
	count, err := b.db.Model(&records).Where("scope = ?", 2).SelectAndCount()
	assert.NoError(t, err)

	fmt.Println("Records size: ", count)

	for i := 0; i < len(records); i++ {
		key, _ := hex.DecodeString(records[i].Key)
		value, _ := hex.DecodeString(records[i].Value)
		b.process(key, value, 1)
	}
}

func TestCreateModels(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()
	b.Publisher = &publisherStub{}

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)
	err := b.db.CreateTable(&Transaction{}, &orm.CreateTableOptions{
		IfNotExists: true,
	})
	require.NoError(t, err)
}
