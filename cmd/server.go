package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/robfig/cron/v3"
	"github.com/slack-go/slack"
	"github.com/webuild-community/core/handler"
	"github.com/webuild-community/core/model"
	"github.com/webuild-community/core/service/command"
	"github.com/webuild-community/core/service/event"
	"github.com/webuild-community/core/service/queue"
	"github.com/webuild-community/core/service/user"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glog "gorm.io/gorm/logger"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	if err := godotenv.Load(); err != nil {
		logger.Error("cannot read .env", zap.Error(err))
	}

	if os.Getenv("SLACK_TOKEN") == "" {
		logger.Panic("missing slack token")
	}

	slackClient := slack.New(os.Getenv("SLACK_TOKEN"))

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("user=%v password=%v dbname=%v host=%v port=5432 sslmode=disable",
			os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_HOST")),
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		Logger: glog.Default.LogMode(glog.Info),
	})
	if err != nil {
		logger.Panic("cannot connect to db", zap.Error(err))
	}
	db.AutoMigrate(&model.User{}, &model.Item{})
	db.AutoMigrate(&model.Transaction{})

	q := queue.NewQueueService()
	commandSvc := command.NewSlackService(logger, db, slackClient)
	eventSvc := event.NewSlackService(logger, db, slackClient)
	userSvc := user.NewPGService(db)

	c := cron.New()
	c.AddFunc("@every 0h5m00s", func() {
		// this will avoid multiple cronjobs may read same data at sometime
		if !q.GetIsConsuming() {
			q.SetIsConsuming(true)
			logger.Info("start handling msg")
			var next interface{}

			for e := q.Consume(); e != nil; e = next {
				if u, ok := e.(*model.User); ok {
					_, isLevelUp, err := userSvc.Update(u.ID, map[string]interface{}{
						"exp": u.Exp,
					})
					if err != nil {
						logger.Error("cannot update user exp", zap.Error(err), zap.String("user_id", u.ID))
					}
					if isLevelUp && u.SlackChannel != "" {
						slackClient.PostMessage(u.SlackChannel, slack.MsgOptionText(fmt.Sprintf("User %v is level up!", u.ID), false))
					}
				}
				next = q.Consume()
				time.Sleep(100 * time.Millisecond)
			}

			logger.Info("end handling success")
			q.SetIsConsuming(false)
		}
	})
	c.Start()

	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	handler.NewEventHandler(e, logger, q, eventSvc, userSvc)
	handler.NewCommandHandler(e, logger, q, commandSvc, userSvc)

	e.Logger.Fatal(e.Start(":8080"))
}
