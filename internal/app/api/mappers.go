package api

import (
	"github.com/insolar/observer/internal/models"
)

func NullableString(s string) *string {
	return &s
}
func NullableInterface(i interface{}) *interface{} {
	return &i
}

func TxToApiTx(txID string, tx models.Transaction) Transaction {
	return Transaction{
		Amount:      tx.Amount,
		Fee:         tx.Fee,
		Index:       0,
		PulseNumber: float32(tx.PulseNumber),
		Status:      string(tx.Status()),
		Timestamp:   0,
		TxID:        txID,
		Type:        string(tx.Type()),
	}
}
