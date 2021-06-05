package transaction

import (
	"gorm.io/gorm"
)

type pg struct {
	db *gorm.DB
}

// NewPGService --
func NewPGService(db *gorm.DB) Service {
	return &pg{db: db}
}
