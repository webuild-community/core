package model

import "time"

type User struct {
	ID             string  `gorm:"size:20;primarykey" json:"user_id"`
	IsAdmin        bool    `gorm:"default:false" json:"is_admin"`
	Exp            int64   `gorm:"default:0" json:"exp"`
	Level          uint    `gorm:"default:1" json:"level"`
	Balance        float64 `gorm:"default:0" json:"balance"`
	GithubUsername string  `json:"github_username"`

	// slack info
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	RealName      string `json:"real_name"`
	DisplayName   string `json:"display_name"`
	TZ            string `json:"tz"`
	ImageOriginal string `json:"image_original"`
	SlackEmail    string `json:"slack_email"`

	WalletAddress string        `json:"wallet_address"`
	Transactions  []Transaction `json:"transactions"`

	// only be used for temporary storing data
	SlackChannel string `gorm:"-" json:"slack_channel"`

	CreatedAt time.Time  `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"default:now()" json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func (User) TableName() string {
	return "user"
}

func (o User) IsLevelUp() bool {
	return o.Exp >= int64(o.Level)*100
}
