package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/slack-go/slack/slackevents"
	"github.com/webuild-community/core/model"
	"github.com/webuild-community/core/service/event"
	"github.com/webuild-community/core/service/queue"
	"github.com/webuild-community/core/service/user"
	"go.uber.org/zap"
)

type EventHandler struct {
	queueSvc queue.Service
	eventSvc event.Service
	userSvc  user.Service
	logger   *zap.Logger
}

func NewEventHandler(e *echo.Echo, logger *zap.Logger, queueSvc queue.Service, eventSvc event.Service, userSvc user.Service) {
	handler := &EventHandler{
		logger:   logger,
		userSvc:  userSvc,
		queueSvc: queueSvc,
		eventSvc: eventSvc,
	}

	e.POST("/slack/events", handler.events)
}

func (h *EventHandler) events(c echo.Context) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		h.logger.Error("cannot read request body", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	event, err := h.eventSvc.Verify(c.Request().Header, body)
	if err != nil {
		h.logger.Error("cannot verify event", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	eventsAPIEvent, ok := event.(slackevents.EventsAPIEvent)
	if !ok {
		h.logger.Error("cannot parse event")
		return c.NoContent(http.StatusInternalServerError)
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			h.logger.Error("cannot unmarshal body", zap.Error(err))
			return c.NoContent(http.StatusInternalServerError)
		}
		c.Response().Header().Set("Content-Type", "text")
		return c.HTMLBlob(http.StatusOK, []byte(r.Challenge))
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		exp := 1

		switch ev := innerEvent.Data.(type) {
		// case *slackevents.AppMentionEvent:
		// 	h.slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))

		case *slackevents.MessageEvent:
			if ev.BotID == ev.User {
				break
			}
			h.logger.Info("received event", zap.String("user_id", ev.User), zap.String("event", "MessageEvent"))

			switch ev.Text {
			case "$profile":
				if err := h.eventSvc.Profile(ev.Channel, ev.User); err != nil {
					h.logger.Error("cannot process $profile event", zap.Error(err))
				}
				return c.NoContent(http.StatusOK)

			case "$register":
				if err := h.eventSvc.Register(ev.User); err != nil {
					h.logger.Error("cannot process $register event", zap.Error(err))
				}
				return c.NoContent(http.StatusOK)

			case "$top":
				if err := h.eventSvc.Top(ev.Channel); err != nil {
					h.logger.Error("cannot process $top event", zap.Error(err))
				}
				return c.NoContent(http.StatusOK)

			case "$drop":
				if err := h.eventSvc.Drop(ev.User); err != nil {
					h.logger.Error("cannot process $drop event", zap.Error(err))
				}
				return c.NoContent(http.StatusOK)

			}

			if len(ev.Text) > 50 {
				exp++
			}

			if err := h.queueSvc.Add(&model.User{ID: ev.User, Exp: int64(exp), SlackChannel: ev.Channel, CreatedAt: time.Now()}); err != nil {
				h.logger.Error("cannot add MessageEvent to queue", zap.Error(err))
			}

		case *slackevents.ReactionAddedEvent:
			if ev.ItemUser == ev.User {
				break
			}
			h.logger.Info("received event", zap.String("user_id", ev.User), zap.String("event", "ReactionAddedEvent"))

			if err := h.queueSvc.Add(&model.User{ID: ev.ItemUser, Exp: int64(exp), CreatedAt: time.Now()}); err != nil {
				h.logger.Error("cannot add ReactionAddedEvent to queue", zap.Error(err))
			}
			if err := h.queueSvc.Add(&model.User{ID: ev.User, Exp: int64(exp), CreatedAt: time.Now()}); err != nil {
				h.logger.Error("cannot add ReactionAddedEvent to queue", zap.Error(err))
			}

		case *slackevents.ReactionRemovedEvent:
			if ev.ItemUser == ev.User {
				break
			}
			h.logger.Info("received event", zap.String("user_id", ev.User), zap.String("event", "ReactionRemovedEvent"))

			if err := h.queueSvc.Add(&model.User{ID: ev.ItemUser, Exp: -1, CreatedAt: time.Now()}); err != nil {
				h.logger.Error("cannot add ReactionRemovedEvent to queue", zap.Error(err))
			}
			if err := h.queueSvc.Add(&model.User{ID: ev.User, Exp: -1, CreatedAt: time.Now()}); err != nil {
				h.logger.Error("cannot add ReactionRemovedEvent to queue", zap.Error(err))
			}

		}

	}

	return c.NoContent(http.StatusOK)
}
