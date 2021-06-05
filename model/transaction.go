package model

import (
	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model
	UserID string  `gorm:"not null" json:"user_id"`
	ItemID uint    `gorm:"not null" json:"item_id"`
	Price  float64 `gorm:"not null" json:"quantity"`
}

func (Transaction) TableName() string {
	return "transaction"
}
