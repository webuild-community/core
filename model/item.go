package model

import (
	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	Name         string        `gorm:"not null" json:"name"`
	Quantity     uint          `gorm:"default:0" json:"quantity"`
	Price        float64       `gorm:"default:0" json:"price"`
	Transactions []Transaction `json:"transactions"`
}

func (Item) TableName() string {
	return "item"
}
