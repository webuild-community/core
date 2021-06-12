package user

import (
	"github.com/webuild-community/core/model"
	"gorm.io/gorm"
)

type pg struct {
	db *gorm.DB
}

// NewPGService --
func NewPGService(db *gorm.DB) Service {
	return &pg{db: db}
}

func (s *pg) Find(id string) (model.User, error) {
	var user model.User
	return user, s.db.First(&user, "id = ?", id).Error
}

func (s *pg) Update(id string, changes map[string]interface{}) (model.User, bool, error) {
	user := model.User{}
	if err := s.db.FirstOrCreate(&user, map[string]interface{}{"id": id}).Error; err != nil {
		return user, false, err
	}

	isLevelUp := false
	if exp, ok := changes["exp"].(int64); ok {
		user.Exp += exp
		isLevelUp = user.IsLevelUp()
		changes["exp"] = user.Exp
		if isLevelUp {
			changes["level"] = user.Level + 1
		}
	}

	return user, isLevelUp, s.db.Model(&user).Updates(changes).Error
}
