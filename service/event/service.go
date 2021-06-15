package event

import (
	"net/http"
)

type Service interface {
	Verify(header http.Header, body []byte) (interface{}, error)
	Profile() error
	Register(userID string) error
	Top() error
	Drop() error
	Redeem() error
}
