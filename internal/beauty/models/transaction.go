package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type InsTransaction struct {
	gorm.Model
	TxID          string `gorm:"column:tx_id;primary_key"`
	Amount        string
	Fee           string
	TimeStamp     uint `gorm:"column:timestamp"`
	Pulse         uint
	Status        string
	ReferenceFrom string `gorm:"column:reference_from"`
	ReferenceTo   string `gorm:"column:reference_to"`
}

func (InsTransaction) TableName() string {
	return "transactions"
}
