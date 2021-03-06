package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo"
	"github.com/slack-go/slack"
	"github.com/webuild-community/core/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthorizeHandler struct {
	clientID     string
	clientSecret string
	logger       *zap.Logger
	db           *gorm.DB
	slack        *slack.Client
}

func NewAuthorizeHandler(
	e *echo.Echo,
	logger *zap.Logger,
	db *gorm.DB,
	slack *slack.Client,
) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if len(clientID) == 0 {
		logger.Fatal("GITHUB_CLIENT_ID is not set")
	}
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if len(clientSecret) == 0 {
		logger.Fatal("GITHUB_CLIENT_SECRET is not set")
	}

	handler := &AuthorizeHandler{
		clientID:     os.Getenv("GITHUB_CLIENT_ID"),
		clientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		logger:       logger,
		db:           db,
		slack:        slack,
	}

	e.GET("/callback/github/auth", handler.handleGithubCallback)
}

type (
	GetAccessTokenReq struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Code         string `json:"code"`
		State        string `json:"state"`
	}

	GetAccessTokenResp struct {
		AccessToken           string `json:"access_token"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
		Scope                 string `json:"scope"`
		TokenType             string `json:"token_type"`
	}

	GithubUser struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
		Name              string `json:"name"`
		Company           string `json:"company"`
		Blog              string `json:"blog"`
		Location          string `json:"location"`
		Email             string `json:"email"`
		Hireable          bool   `json:"hireable"`
		Bio               string `json:"bio"`
		TwitterUsername   string `json:"twitter_username"`
		PublicRepos       int    `json:"public_repos"`
		PublicGists       int    `json:"public_gists"`
		Followers         int    `json:"followers"`
		Following         int    `json:"following"`
		PrivateGists      int    `json:"private_gists"`
		TotalPrivateRepos int    `json:"total_private_repos"`
		OwnedPrivateRepos int    `json:"owned_private_repos"`
		Collaborators     int    `json:"collaborators"`
	}
)

func (h *AuthorizeHandler) handleGithubCallback(ctx echo.Context) error {
	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")

	user := model.User{}
	if err := h.db.
		Where(&model.User{ID: state}).
		Find(&user).Error; err != nil {
		h.logger.Error("get user failed", zap.Error(err))
		return err
	}

	if len(user.GithubUsername) > 0 {
		return h.sendSlackRegiterSuccessMsg(&user)
	}

	client := http.Client{}
	body, _ := json.Marshal(&GetAccessTokenReq{
		ClientID:     h.clientID,
		ClientSecret: h.clientSecret,
		Code:         code,
		State:        state,
	})
	resp, err := client.Post(
		"https://github.com/login/oauth/access_token",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		h.logger.Error("get user access token failed", zap.Error(err))
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("get data from server response failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	rawAccessToken := string(respBody) // Form: access_token=<token>&scope=<scope>&token_type=bearer
	accessToken := rawAccessToken[len("access_token="):strings.Index(rawAccessToken, "&")]

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		h.logger.Error("create request failed", zap.Error(err))
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Set("X-OAuth-Scopes", "user")
	req.Header.Set("X-Accepted-OAuth-Scopes", "user")

	resp, err = client.Do(req)
	if err != nil {
		h.logger.Error("get user access token failed", zap.Error(err))
		return err
	}

	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("get data from server response failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	githubUser := GithubUser{}
	if err := json.Unmarshal(respBody, &githubUser); err != nil {
		h.logger.Error("parse data from github response failed", zap.Error(err))
		return err
	}

	user.GithubUsername = githubUser.Login
	user.GithubBio = githubUser.Bio
	if err := h.db.
		Model(&model.User{}).
		Where(&model.User{ID: user.ID}).
		Updates(&user).Error; err != nil {
		h.logger.Error("update user failed", zap.Error(err))
		return err
	}

	return h.sendSlackRegiterSuccessMsg(&user)
}

func (h *AuthorizeHandler) sendSlackRegiterSuccessMsg(user *model.User) error {
	blockText := slack.NewTextBlockObject(
		"mrkdwn",
		"*Github register*\nWelcome to WeXu, your account has been registered",
		false, false)
	section := slack.NewSectionBlock(blockText, nil, nil)

	channel, _, _, err := h.slack.OpenConversation(&slack.OpenConversationParameters{
		Users:    []string{user.ID},
		ReturnIM: true,
	})
	if err != nil {
		h.logger.Error("open direct message failed", zap.Error(err))
		return err
	}
	if _, _, _, err := h.slack.SendMessage(
		channel.ID,
		slack.MsgOptionBlocks(section),
	); err != nil {
		h.logger.Error("send message failed", zap.Error(err))
		return err
	}

	return nil
}
