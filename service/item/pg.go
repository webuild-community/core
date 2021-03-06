package item

import (
	"context"
	"errors"
	"os"

	"github.com/dstotijn/go-notion"
	"github.com/webuild-community/core/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type pg struct {
	logger       *zap.Logger
	notionClient *notion.Client
	db           *gorm.DB
}

// NewPGService --
func NewPGService(logger *zap.Logger, notionClient *notion.Client, db *gorm.DB) Service {
	return &pg{
		logger:       logger,
		notionClient: notionClient,
		db:           db,
	}
}

func (s *pg) Find(id string) (*model.Item, error) {
	isExpired := false
	items, err := s.notionClient.QueryDatabase(context.Background(), os.Getenv("NOTION_DATABASE_ID"), &notion.DatabaseQuery{Filter: &notion.DatabaseQueryFilter{
		Property: "Expired",
		Checkbox: &notion.CheckboxDatabaseQueryFilter{Equals: &isExpired},
	}})
	if err != nil {
		s.logger.Error("cannot fetch items", zap.Error(err))
		return nil, err
	}

	for _, v := range items.Results {
		if v.ID != id {
			continue
		}

		properties, ok := v.Properties.(notion.DatabasePageProperties)
		if !ok {
			continue
		}

		return &model.Item{
			ID:       id,
			Name:     properties["Name"].Title[0].PlainText,
			Quantity: uint(*properties["Quantity"].Number),
			Redeemed: uint(*properties["Redeemed"].Number),
			Price:    *properties["Price"].Number,
		}, nil
	}

	return nil, errors.New("item not found")
}

func (s *pg) Redeem(itemID, userID string) error {
	s.logger.Info("handling Redeem", zap.String("item_id", itemID))

	item, err := s.Find(itemID)
	if err != nil {
		return err
	}

	txs := []model.Transaction{}
	if err := s.db.Find(&txs, "item_id = ?", itemID).Error; err != nil {
		return err
	}
	if len(txs) == int(item.Quantity) {
		return errors.New("out of stock")
	}

	return s.db.Create(&model.Transaction{
		ItemID: itemID,
		UserID: userID,
		Price:  item.Price,
	}).Error
}
