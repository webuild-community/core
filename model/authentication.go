package model

import "gorm.io/gorm"

type AuthenticationStatus uint

const (
	AuthenticationPending AuthenticationStatus = iota + 1
	AuthenticationSuccessful
	AuthenticationFailed
)

type Provider string

const (
	ProviderGithub Provider = "github"
)

type Authentication struct {
	gorm.Model
	Provider Provider             `json:"provider"`
	State    string               `json:"state"`
	Status   AuthenticationStatus `json:"status"`
}

func (Authentication) TableName() string {
	return "authentication"
}
