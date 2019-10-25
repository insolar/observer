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

func TxToApiTx(txID string, tx models.Transaction) interface{} {
	internalTx := Transaction{
		Amount:      tx.Amount,
		Fee:         tx.Fee,
		Index:       0,
		PulseNumber: float32(tx.PulseNumber),
		Status:      string(tx.Status()),
		Timestamp:   0,
		TxID:        txID,
		Type:        string(tx.Type()),
	}

	switch tx.Type() {
	case models.TTypeMigration:
		return Migration{
			Transaction:         internalTx,
			FromMemberReference: NullableString(string(tx.MemberFromReference)),
			ToDepositReference:  NullableString(string(tx.MigrationsToReference)),
			ToMemberReference:   NullableString(string(tx.MemberToReference)),
			Type:                NullableInterface(tx.Type()),
		}
	case models.TTypeTransfer:
		return Transfer{
			Transaction:         internalTx,
			FromMemberReference: NullableString(string(tx.MemberFromReference)),
			ToMemberReference:   NullableString(string(tx.MemberToReference)),
			Type:                NullableInterface(tx.Type()),
		}
	case models.TTypeRelease:
		return Release{
			Transaction:          internalTx,
			FromDepositReference: NullableString(string(tx.VestingFromReference)),
			ToMemberReference:    NullableString(string(tx.MemberToReference)),
			Type:                 NullableInterface(tx.Type()),
		}
	default:
		return internalTx
	}
}
