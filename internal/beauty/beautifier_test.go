package beauty

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBeautifier_parse(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	b.parse([]byte("000100010307e48bc120c3d42213d9fd45b6a346f117e3edd27140b5583ed1ea"), []byte("a201e201c206de01a00103aa014000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000b2014000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f2010f6e6f6465646f6d61696e5f636f646582024000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000aa01200000000100000000000000000000000000000000000000000000000000000000"), 1)
}

func TestBeautifier_storeTx(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	tx := InsTransaction{TxID: "foo", Amount: "1000", Fee: "100",
		Timestamp: 5555, Pulse: 32323, Status: "SUCCESS", ReferenceFrom: "alpha", ReferenceTo: "beta"}

	assert.NoError(t, b.storeTx(tx))
}

func TestBeautifier_ParseAndStore(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	var records []InsRecord
	count, err := b.db.Model(&records).Limit(3000).Where("scope = ?", 2).SelectAndCount()
	assert.NoError(t, err)

	fmt.Println("Records size: ", count)

	for i := 0; i < len(records); i++ {
		b.parse([]byte(records[i].Key), []byte(records[i].Value), 1)
	}
}
