package event

import (
	"encoding/json"
<<<<<<< HEAD
	"errors"
=======
>>>>>>> 6b28f21 (feat: add Github register service (#12))
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
<<<<<<< HEAD
	ClientID string
	logger   *zap.Logger
	db       *gorm.DB
	client   *slack.Client
=======
	githubClientID string
	logger         *zap.Logger
	db             *gorm.DB
	client         *slack.Client
>>>>>>> 6b28f21 (feat: add Github register service (#12))
}

// NewSlackService --
func NewSlackService(logger *zap.Logger, db *gorm.DB, client *slack.Client) Service {
<<<<<<< HEAD
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if len(clientID) == 0 {
=======
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	if len(githubClientID) == 0 {
>>>>>>> 6b28f21 (feat: add Github register service (#12))
		logger.Fatal("GITHUB_CLIENT_ID is not set")
	}

	return &slackSvc{
<<<<<<< HEAD
		ClientID: clientID,
		logger:   logger,
		db:       db,
		client:   client,
=======
		githubClientID: githubClientID,
		logger:         logger,
		db:             db,
		client:         client,
>>>>>>> 6b28f21 (feat: add Github register service (#12))
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

<<<<<<< HEAD
func (s *slackSvc) Register(userID, channel string) error {
=======
func (s *slackSvc) Register(userID string) error {
>>>>>>> 6b28f21 (feat: add Github register service (#12))
	slackProfile, err := s.client.GetUserProfile(&slack.GetUserProfileParameters{
		UserID: userID,
	})
	if err != nil {
		s.logger.Error("failed to get user info", zap.Error(err))
		return err
	}

<<<<<<< HEAD
	user := model.User{}
	res := s.db.Joins(`join authentication on "authentication"."id" = "user"."authentication_id"`).
		Where(`"user"."id" = ? and "authentication"."status" = ?`, userID, model.AuthenticationSuccessful).
		Find(&user)
	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
=======
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
>>>>>>> 6b28f21 (feat: add Github register service (#12))
		s.logger.Error("failed to get user from db", zap.Error(err))
		return err
	}

<<<<<<< HEAD
	// User is already exists
	if res.RowsAffected > 0 {
		if _, _, _, err := s.client.SendMessage(channel, slack.MsgOptionText(
=======
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
>>>>>>> 6b28f21 (feat: add Github register service (#12))
			"*Account registered*\nWelcome to WeXu, your account has been registered!",
			true,
		)); err != nil {
			s.logger.Error("send message failed", zap.Error(err))
			return err
		}

		return nil
	}

<<<<<<< HEAD
	state := uuid.NewString()
	link := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=user", s.ClientID, state)
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
=======
	link := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=user", s.githubClientID, userID)
	text := fmt.Sprintf("*Github register*\nWelcome to WeXu, please register your account\n *<%s|Register WeXu account with Github>*", link)
	blockText := slack.NewTextBlockObject("mrkdwn", text, false, true)
	section := slack.NewSectionBlock(blockText, nil, nil)

	if _, _, _, err := s.client.SendMessage(channel.ID, slack.MsgOptionBlocks(section)); err != nil {
		s.logger.Error("send message failed", zap.Error(err))
>>>>>>> 6b28f21 (feat: add Github register service (#12))
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
