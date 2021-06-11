package user

import "github.com/webuild-community/core/model"

type Service interface {
	Find(id string) (model.User, bool, error)
	Update(id string, changes map[string]interface{}) (model.User, bool, error)
}
