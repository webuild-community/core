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

func (s *slackSvc) Profile(channelID, userID string) error {
	var user model.User
	err := s.db.First(&user, "id = ?", userID).Error
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

func (s *slackSvc) Top(channelID string) error {
	users := make([]model.User, 0)
	err := s.db.Model(model.User{}).Where("level > ?", 1).Order("level DESC").Limit(10).Find(&users).Error
	if err != nil {
		return err
	}

	blocks := buildBlockUserTopMessage(users)
	if len(blocks) == 0 {
		if _, _, _, err := s.client.SendMessage(channelID, slack.MsgOptionText("No users reached the top", false)); err != nil {
			s.logger.Error("Send message failed", zap.Error(err))
			return err
		}
	}
	if _, _, _, err = s.client.SendMessage(channelID, slack.MsgOptionBlocks(blocks...)); err != nil {
		s.logger.Error("Send message failed", zap.Error(err))
		return err
	}

	return nil
}

func buildBlockUserTopMessage(users []model.User) []slack.Block {
	divider := slack.NewDividerBlock()
	blocks := make([]slack.Block, 0)

	header := slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "*Top Users*", false, false), nil, nil)
	blocks = append(blocks, header)
	blocks = append(blocks, divider)

	for index, user := range users {
		imageAccessory := slack.NewAccessory(slack.NewImageBlockElement(user.ImageOriginal, user.GithubUsername))
		userInfoBlock := slack.NewTextBlockObject("mrkdwn", buildUserInfoMessage(user), false, false)
		section := slack.NewSectionBlock(userInfoBlock, nil, imageAccessory)
		blocks = append(blocks, section)
		if index != len(users)-1 {
			blocks = append(blocks, divider)
		}
	}
	return blocks
}

func buildUserInfoMessage(user model.User) string {
	return fmt.Sprintf("*User*: %s\n*Level*: %d\n*Balance*: %0.1f", user.GithubUsername, user.Level, user.Balance)
}

func (s *slackSvc) Drop() error {
	s.logger.Info("handling drop")
	return nil
}

func (s *slackSvc) Redeem() error {
	s.logger.Info("handling redeem")
	return nil
}
