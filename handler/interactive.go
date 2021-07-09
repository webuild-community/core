package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/labstack/echo"
	"github.com/slack-go/slack"
	"github.com/webuild-community/core/service/item"
	"github.com/webuild-community/core/service/queue"
	"github.com/webuild-community/core/service/user"
	"go.uber.org/zap"
)

type InteractiveHandler struct {
	logger   *zap.Logger
	queueSvc queue.Service
	userSvc  user.Service
	itemSvc  item.Service
}

func NewInteractiveHandler(e *echo.Echo, logger *zap.Logger, queueSvc queue.Service, userSvc user.Service, itemSvc item.Service) {
	handler := &InteractiveHandler{
		logger:   logger,
		queueSvc: queueSvc,
		userSvc:  userSvc,
		itemSvc:  itemSvc,
	}

	e.POST("/slack/interactives", handler.interactives)
}

func (h *InteractiveHandler) interactives(c echo.Context) error {
	buf, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	jsonStr, err := url.QueryUnescape(string(buf)[8:])
	if err != nil {
		h.logger.Error("failed to unespace request body", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	var message slack.AttachmentActionCallback
	if err := json.Unmarshal([]byte(jsonStr), &message); err != nil {
		h.logger.Error("failed to decode json message from slack", zap.Any("jsonStr", jsonStr))
		return c.NoContent(http.StatusInternalServerError)
	}

	// Only accept message from slack with valid token
	if message.Token != os.Getenv("SLACK_VERIFICATION_TOKEN") {
		h.logger.Error("invalid token", zap.Any("token", message.Token))
		return c.NoContent(http.StatusUnauthorized)
	}

	if len(message.ActionCallback.BlockActions) == 0 {
		return c.NoContent(http.StatusBadRequest)
	}

	switch message.ActionCallback.BlockActions[0].Text.Text {
	case "Redeem":
		itemID := message.ActionCallback.BlockActions[0].Value
		if err := h.itemSvc.Redeem(itemID, message.User.ID); err != nil {
			h.logger.Error("cannot redeem item", zap.Error(err), zap.String("item_id", itemID), zap.String("user_id", message.User.ID))
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.NoContent(http.StatusOK)
	}

	return c.NoContent(http.StatusBadRequest)
}
