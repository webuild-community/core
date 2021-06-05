package event

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type slackSvc struct {
	logger *zap.Logger
	db     *gorm.DB
	client *slack.Client
}

// NewSlackService --
func NewSlackService(logger *zap.Logger, db *gorm.DB, client *slack.Client) Service {
	return &slackSvc{
		logger: logger,
		db:     db,
		client: client,
	}
}

func (s *slackSvc) Verify(header http.Header, body []byte) (interface{}, error) {
	sv, err := slack.NewSecretsVerifier(header, os.Getenv("SLACK_SIGNING_SECRET"))
	if err != nil {
		s.logger.Error("cannot init secret verifier", zap.Error(err))
		return nil, err
	}
	if _, err := sv.Write(body); err != nil {
		s.logger.Error("cannot write body", zap.Error(err))
		return nil, err
	}
	if err := sv.Ensure(); err != nil {
		s.logger.Error("cannot ensure", zap.Error(err))
		return nil, err
	}

	return slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
}

func (s *slackSvc) Profile() error {
	s.logger.Info("handling profile")
	return nil
}

func (s *slackSvc) Register() error {
	s.logger.Info("handling register")
	return nil
}

func (s *slackSvc) Top() error {
	s.logger.Info("handling top")
	return nil
}

func (s *slackSvc) Drop() error {
	s.logger.Info("handling drop")
	return nil
}

func (s *slackSvc) Redeem() error {
	s.logger.Info("handling redeem")
	return nil
}
