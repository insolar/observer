package models

import "github.com/jinzhu/gorm"

type InsMember struct {
	gorm.Model
	Reference        string
	Balance          string
	MigrationAddress string
}

func (InsMember) TableName() string {
	return "members"
}
