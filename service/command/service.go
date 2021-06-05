package command

import "net/http"

type Service interface {
	Verify(r *http.Request) (interface{}, error)
	Sync() error
}
