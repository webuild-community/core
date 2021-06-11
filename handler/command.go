package handler

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/slack-go/slack"
	"github.com/webuild-community/core/service/command"
	"github.com/webuild-community/core/service/queue"
	"github.com/webuild-community/core/service/user"
	"go.uber.org/zap"
)

type CommandHandler struct {
	queueSvc   queue.Service
	commandSvc command.Service
	userSvc    user.Service
	logger     *zap.Logger
}

func NewCommandHandler(e *echo.Echo, logger *zap.Logger, queueSvc queue.Service, commandSvc command.Service, userSvc user.Service) {
	handler := &CommandHandler{
		logger:     logger,
		userSvc:    userSvc,
		queueSvc:   queueSvc,
		commandSvc: commandSvc,
	}

	e.POST("/slack/commands", handler.commands)
}

func (h *CommandHandler) commands(c echo.Context) error {
	cmd, err := h.commandSvc.Verify(c.Request())
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	s, ok := cmd.(slack.SlashCommand)
	if !ok {
		return c.NoContent(http.StatusInternalServerError)
	}

	user, exist, err := h.userSvc.Find(s.UserID)
	if err != nil {
		h.logger.Error("cannot find user", zap.Error(err), zap.String("user_id", s.UserID))
		return c.NoContent(http.StatusNotFound)
	}
	if !exist {
		h.logger.Error("User doest not exist", zap.Error(err), zap.String("user_id", s.UserID))
		return c.NoContent(http.StatusNotFound)
	}

	switch s.Command {
	case "/sync":
		if !user.IsAdmin {
			return c.String(http.StatusForbidden, "Forbidden")
		}

		if err := h.commandSvc.Sync(); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.String(http.StatusOK, "Synced from airtable successful")

	}

	return c.NoContent(http.StatusInternalServerError)
}
