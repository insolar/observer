package beauty

import (
	"context"
	"github.com/insolar/insolar/ledger/heavy/sequence"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBeautifier_parse(t *testing.T) {
	ctx := context.Background()
	b := NewBeautifier()

	assert.NoError(t, b.Init(ctx))
	assert.NoError(t, b.Start(ctx))
	defer b.Stop(ctx)

	b.parse(sequence.Item{}, 0)
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
