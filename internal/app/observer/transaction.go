// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package observer

import (
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/models"
)

type TxRegister struct {
	TransactionID        insolar.Reference
	Type                 models.TransactionType
	PulseNumber          int64
	RecordNumber         int64
	MemberFromReference  []byte
	MemberToReference    []byte
	DepositToReference   []byte
	DepositFromReference []byte
	Amount               string
}

func (t *TxRegister) Validate() error {
	if t.TransactionID.IsEmpty() {
		return errors.New("TransactionID should not be empty")
	}
	if len(t.Type) == 0 {
		return errors.New("Type should not be empty")
	}
	if t.Type == models.TTypeUnknown {
		return errors.New("Type should not be Unknown")
	}
	if t.PulseNumber == 0 {
		return errors.New("PulseNumber should not be zero")
	}
	if len(t.Amount) == 0 {
		return errors.New("Amount should not be empty")
	}
	return nil
}

type TxFailed struct {
	FinishPulseNumber  int64
	FinishRecordNumber int64
}

type TxResult struct {
	TransactionID insolar.Reference
	Fee           string
	Failed        *TxFailed
}

func (t *TxResult) Validate() error {
	if t.TransactionID.IsEmpty() {
		return errors.New("TransactionID should not be empty")
	}
	if t.Failed == nil && len(t.Fee) == 0 {
		return errors.New("Fee should not be empty")
	}
	return nil
}

type TxSagaResult struct {
	TransactionID      insolar.Reference
	FinishSuccess      bool
	FinishPulseNumber  int64
	FinishRecordNumber int64
}

func (t *TxSagaResult) Validate() error {
	if t.TransactionID.IsEmpty() {
		return errors.New("TransactionID should not be empty")
	}
	if t.FinishPulseNumber == 0 {
		return errors.New("FinishPulseNumber should not be zero")
	}
	return nil
}

type TxDepositTransferUpdate struct {
	TransactionID        insolar.Reference
	DepositFromReference []byte
}

func (t *TxDepositTransferUpdate) Validate() error {
	if t.TransactionID.IsEmpty() {
		return errors.New("TransactionID should not be empty")
	}
	if len(t.DepositFromReference) == 0 {
		return errors.New("DepositFromReference should not be empty")
	}
	return nil
}
