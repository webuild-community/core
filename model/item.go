package model

import "time"

type Item struct {
	ID       string  `json:"id"`
	Name     string  `gorm:"not null" json:"name"`
	Quantity uint    `gorm:"default:0" json:"quantity"`
	Redeemed uint    `gorm:"default:0" json:"redeemed"`
	Price    float64 `gorm:"default:0" json:"price"`
	// Transactions []Transaction `json:"transactions"`
	CreatedAt time.Time  `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"default:now()" json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func (Item) TableName() string {
	return "item"
}
