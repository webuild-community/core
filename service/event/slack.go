package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/webuild-community/core/model"
	"github.com/webuild-community/core/service/user"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type slackSvc struct {
	githubClientID string
	logger         *zap.Logger
	db             *gorm.DB
	client         *slack.Client
	userSvc        user.Service
}

// NewSlackService --
func NewSlackService(logger *zap.Logger, db *gorm.DB, client *slack.Client, userSvc user.Service) Service {
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	if len(githubClientID) == 0 {
		logger.Fatal("GITHUB_CLIENT_ID is not set")
	}
	return &slackSvc{
		githubClientID: githubClientID,
		logger:         logger,
		db:             db,
		client:         client,
		userSvc:        userSvc,
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

func (s *slackSvc) Profile(channelID, userID string) error {
	user, err := s.userSvc.Find(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.client.PostMessage(channelID, slack.MsgOptionText("Please type `$register` command first", false))
			return nil
		}
		s.client.PostMessage(channelID, slack.MsgOptionText("Please try again later", false))
		return err
	}
	payload := fmt.Sprintf("Exp: `%d`, level: %d", user.Exp, user.Level)
	_, _, err = s.client.PostMessage(channelID, slack.MsgOptionText(payload, false))
	return err
}

func (s *slackSvc) Register(userID string) error {
	slackProfile, err := s.client.GetUserProfile(&slack.GetUserProfileParameters{
		UserID: userID,
	})
	if err != nil {
		s.logger.Error("failed to get user info", zap.Error(err))
		return err
	}

	user := model.User{
		ID:            userID,
		FirstName:     slackProfile.FirstName,
		LastName:      slackProfile.LastName,
		RealName:      slackProfile.RealName,
		DisplayName:   slackProfile.DisplayName,
		ImageOriginal: slackProfile.Image48,
		SlackEmail:    slackProfile.Email,
	}
	if err := s.db.
		Where(`"user"."id" = ?`, userID).
		FirstOrCreate(&user).Error; err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("failed to get user from db", zap.Error(err))
		return err
	}

	channel, _, _, err := s.client.OpenConversation(&slack.OpenConversationParameters{
		Users:    []string{userID},
		ReturnIM: true,
	})
	if err != nil {
		s.logger.Error("open direct message failed", zap.Error(err))
		return err
	}

	// User is already exists
	if len(user.GithubUsername) > 0 {
		if _, _, _, err := s.client.SendMessage(channel.ID, slack.MsgOptionText(
			"*Account registered*\nWelcome to WeXu, your account has been registered!",
			true,
		)); err != nil {
			s.logger.Error("send message failed", zap.Error(err))
			return err
		}

		return nil
	}

	link := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=user", s.githubClientID, userID)
	text := fmt.Sprintf("*Github register*\nWelcome to WeXu, please register your account\n *<%s|Register WeXu account with Github>*", link)
	blockText := slack.NewTextBlockObject("mrkdwn", text, false, true)
	section := slack.NewSectionBlock(blockText, nil, nil)

	if _, _, _, err := s.client.SendMessage(channel.ID, slack.MsgOptionBlocks(section)); err != nil {
		s.logger.Error("send message failed", zap.Error(err))
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
