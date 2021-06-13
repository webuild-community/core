package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/webuild-community/core/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type slackSvc struct {
	githubClientID string
	logger         *zap.Logger
	db             *gorm.DB
	client         *slack.Client
}

// NewSlackService --
func NewSlackService(logger *zap.Logger, db *gorm.DB, client *slack.Client) Service {
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	if len(githubClientID) == 0 {
		logger.Fatal("GITHUB_CLIENT_ID is not set")
	}

	return &slackSvc{
		githubClientID: githubClientID,
		logger:         logger,
		db:             db,
		client:         client,
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

func (s *slackSvc) Register(userID, channel string) error {
	slackProfile, err := s.client.GetUserProfile(&slack.GetUserProfileParameters{
		UserID: userID,
	})
	if err != nil {
		s.logger.Error("failed to get user info", zap.Error(err))
		return err
	}

	user := model.User{}
	res := s.db.Joins(`join authentication on "authentication"."id" = "user"."authentication_id"`).
		Where(`"user"."id" = ? and "authentication"."status" = ?`, userID, model.AuthenticationSuccessful).
		Find(&user)
	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to get user from db", zap.Error(err))
		return err
	}

	// User is already exists
	if res.RowsAffected > 0 {
		if _, _, _, err := s.client.SendMessage(channel, slack.MsgOptionText(
			"*Account registered*\nWelcome to WeXu, your account has been registered!",
			true,
		)); err != nil {
			s.logger.Error("send message failed", zap.Error(err))
			return err
		}

		return nil
	}

	state := uuid.NewString()
	link := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=user", s.githubClientID, state)
	text := fmt.Sprintf("*Github register*\nWelcome to WeXu, please register your account\n *<%s|Register>*", link)
	blockText := slack.NewTextBlockObject("mrkdwn", text, false, true)
	accessory := slack.NewImageBlockElement("https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png", "github thumbnail")
	section := slack.NewSectionBlock(blockText, nil, slack.NewAccessory(accessory))
	if _, _, _, err := s.client.SendMessage(channel, slack.MsgOptionBlocks(section)); err != nil {
		s.logger.Error("send message failed", zap.Error(err))
		return err
	}

	if err := s.db.Create(&model.User{
		ID:            userID,
		FirstName:     slackProfile.FirstName,
		LastName:      slackProfile.LastName,
		RealName:      slackProfile.RealName,
		DisplayName:   slackProfile.DisplayName,
		ImageOriginal: slackProfile.Image48,
		SlackEmail:    slackProfile.Email,
		SlackChannel:  channel,
		Authentication: model.Authentication{
			State:    state,
			Provider: model.ProviderGithub,
			Status:   model.AuthenticationPending,
		},
	}).Error; err != nil {
		s.logger.Error("create user failed", zap.Error(err))
		return err
	}

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
