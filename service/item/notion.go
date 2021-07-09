package item

import (
	"context"
	"errors"
	"os"

	"github.com/dstotijn/go-notion"
	"github.com/webuild-community/core/model"
	"go.uber.org/zap"
)

type notionSvc struct {
	logger       *zap.Logger
	notionClient *notion.Client
}

// NewNotionService --
func NewNotionService(logger *zap.Logger, notionClient *notion.Client) Service {
	return &notionSvc{
		logger:       logger,
		notionClient: notionClient,
	}
}

func (s *notionSvc) Find(id string) (*model.Item, error) {
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

func (s *notionSvc) Redeem(itemID, userID string) error {
	s.logger.Info("handling Redeem", zap.String("item_id", itemID))

	item, err := s.Find(itemID)
	if err != nil {
		return err
	}

	redeemed := float64(item.Redeemed + 1)
	s.notionClient.UpdatePageProps(context.Background(),
		itemID,
		notion.UpdatePageParams{DatabasePageProperties: &notion.DatabasePageProperties{
			"Redeemed": notion.DatabasePageProperty{
				Type:   "number",
				Number: &redeemed,
			}}},
	)

	return nil
}
