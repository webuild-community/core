package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/dstotijn/go-notion"
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
	slackClient    *slack.Client
	notionClient   *notion.Client
}

// NewSlackService --
func NewSlackService(logger *zap.Logger, db *gorm.DB, slackClient *slack.Client, notionClient *notion.Client) Service {
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	if len(githubClientID) == 0 {
		logger.Fatal("GITHUB_CLIENT_ID is not set")
	}
	return &slackSvc{
		githubClientID: githubClientID,
		logger:         logger,
		db:             db,
		slackClient:    slackClient,
		notionClient:   notionClient,
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
			s.slackClient.PostMessage(channelID, slack.MsgOptionText("Please type `$register` command first", false))
			return nil
		}
		s.slackClient.PostMessage(channelID, slack.MsgOptionText("Please try again later", false))
		return err
	}

	payload := fmt.Sprintf("Exp: `%d`, level: %d", user.Exp, user.Level)
	_, _, err = s.slackClient.PostMessage(channelID, slack.MsgOptionText(payload, false))
	return err
}

func (s *slackSvc) dmUser(userID string, options ...slack.MsgOption) error {
	channel, _, _, err := s.slackClient.OpenConversation(&slack.OpenConversationParameters{
		Users:    []string{userID},
		ReturnIM: true,
	})
	if err != nil {
		s.logger.Error("open direct message failed", zap.Error(err))
		return err
	}

	if _, _, _, err := s.slackClient.SendMessage(channel.ID, options...); err != nil {
		s.logger.Error("send message failed", zap.Error(err))
		return err
	}
	return nil
}

func (s *slackSvc) Register(userID string) error {
	slackProfile, err := s.slackClient.GetUserProfile(&slack.GetUserProfileParameters{
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

	// User is already exists
	if user.GithubUsername != "" {
		return s.dmUser(userID, slack.MsgOptionText(
			"*Account registered*\nWelcome to WeXu, your account has been registered!",
			true,
		))
	}

	link := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=user", s.githubClientID, userID)
	text := fmt.Sprintf("*Github register*\nWelcome to WeXu, please register your account\n *<%s|Register WeXu account with Github>*", link)
	blockText := slack.NewTextBlockObject("mrkdwn", text, false, true)
	section := slack.NewSectionBlock(blockText, nil, nil)

	return s.dmUser(userID, slack.MsgOptionBlocks(section))
}

func (s *slackSvc) Top(channelID string) error {
	users := make([]model.User, 0)
	err := s.db.Model(model.User{}).Order("exp DESC").Limit(10).Find(&users).Error
	if err != nil {
		return err
	}
	if len(users) == 0 {
		if err := s.dmUser(channelID, slack.MsgOptionText("No users reached the top", false)); err != nil {
			return err
		}
	}

	blocks := buildBlockUserTopMessage(users)
	if err := s.dmUser(channelID, slack.MsgOptionBlocks(blocks...)); err != nil {
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

func (s *slackSvc) Drop(userID string) error {
	isExpired := false
	items, err := s.notionClient.QueryDatabase(context.Background(), os.Getenv("NOTION_DATABASE_ID"), &notion.DatabaseQuery{Filter: &notion.DatabaseQueryFilter{
		Property: "Expired",
		Checkbox: &notion.CheckboxDatabaseQueryFilter{Equals: &isExpired},
	}})
	if err != nil {
		s.logger.Error("cannot fetch items", zap.Error(err))
	}

	pretext := fmt.Sprintf("*Drop Items*\n")
	blockPretext := slack.NewTextBlockObject("mrkdwn", pretext, false, true)
	sectionBlockPretext := slack.NewSectionBlock(blockPretext, nil, nil)

	attachment := slack.Attachment{
		Color: "#3AA3E3",
	}

	blockset := []slack.Block{}
	for _, v := range items.Results {
		properties, ok := v.Properties.(notion.DatabasePageProperties)
		if !ok {
			continue
		}

		if len(properties["Name"].Title) == 0 ||
			properties["Redeemed"].Number == nil ||
			properties["Quantity"].Number == nil ||
			properties["Price"].Number == nil {
			continue
		}

		if *properties["Redeemed"].Number >= *properties["Quantity"].Number {
			continue
		}

		text := fmt.Sprintf("*%v* (%v/%v)\n", properties["Name"].Title[0].PlainText, *properties["Redeemed"].Number, *properties["Quantity"].Number)
		if len(properties["Description"].Title) > 0 {
			text += fmt.Sprintf("Description: %v\n", properties["Description"].Title[0].PlainText)
		}
		text += fmt.Sprintf("%v RDF", *properties["Price"].Number)

		redeemBtnTxt := slack.NewTextBlockObject("plain_text", "Redeem", false, false)
		redeemButton := slack.NewButtonBlockElement("", v.ID, redeemBtnTxt)
		redeemButton.Style = "primary"
		block := slack.NewTextBlockObject("mrkdwn", text, false, true)
		sectionBlock := slack.NewSectionBlock(block, nil, slack.NewAccessory(redeemButton))
		blockset = append(blockset, sectionBlock)
	}
	attachment.Blocks = slack.Blocks{BlockSet: blockset}

	return s.dmUser(userID, slack.MsgOptionBlocks(sectionBlockPretext), slack.MsgOptionAttachments(attachment))
}
