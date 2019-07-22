package models

import "github.com/jinzhu/gorm"

type InsFee struct {
	gorm.Model
}

func (InsFee) TableName() string {
	return "fees"
}
