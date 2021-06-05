package command

import (
	"net/http"
	"os"

	"github.com/slack-go/slack"
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

func (s *slackSvc) Sync() error {
	s.logger.Info("handling sync")
	return nil
}

func (s *slackSvc) Verify(r *http.Request) (interface{}, error) {
	cmd, err := slack.SlashCommandParse(r)
	if err != nil {
		return nil, err
	}

	if !cmd.ValidateToken(os.Getenv("SLACK_VERIFICATION_TOKEN")) {
		return nil, err
	}

	return cmd, nil
}
