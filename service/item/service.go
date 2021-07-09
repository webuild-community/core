package item

import "github.com/webuild-community/core/model"

type Service interface {
	Find(id string) (*model.Item, error)
	Redeem(itemID, userID string) error
}
