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

func (s *pg) Find(id string) (model.User, bool, error) {
	var user model.User
	err := s.db.First(&user, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return user, false, nil
	}
	return user, true, err
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
