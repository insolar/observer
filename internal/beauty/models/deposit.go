package models

import "github.com/jinzhu/gorm"

type InsDeposit struct {
	gorm.Model
	Timestamp       uint
	HoldReleaseDate uint
	Amount          string
	Bonus           string
	EthHash         string
	Status          string
	MemberID        uint
}

func (InsDeposit) TableName() string {
	return "deposits"
}
