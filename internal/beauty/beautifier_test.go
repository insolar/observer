package beauty

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

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

	b.parse(key, value, 1)
}

func TestBeautifier_storeTx(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()
	b.Publisher = &publisherStub{}

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	tx := Transaction{TxID: "foo", Amount: "1000", Fee: "100",
		Timestamp: 5555, Pulse: 32323, Status: "SUCCESS", ReferenceFrom: "alpha", ReferenceTo: "beta"}

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
		b.parse(key, value, 1)
	}
}
