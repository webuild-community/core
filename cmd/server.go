package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dstotijn/go-notion"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/robfig/cron/v3"
	"github.com/slack-go/slack"
	"github.com/webuild-community/core/handler"
	"github.com/webuild-community/core/model"
	"github.com/webuild-community/core/service/command"
	"github.com/webuild-community/core/service/event"
	"github.com/webuild-community/core/service/item"
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

	slackClient := slack.New(os.Getenv("SLACK_TOKEN"), slack.OptionDebug(true))
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("user=%v password=%v dbname=%v host=%v port=%v sslmode=disable",
			os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		Logger: glog.Default.LogMode(glog.Info),
	})
	if err != nil {
		logger.Panic("cannot connect to db", zap.Error(err))
	}

	notionClient := notion.NewClient(os.Getenv("NOTION_SECRET_KEY"))

	if err := db.AutoMigrate(
		&model.User{},
		&model.Item{},
		&model.Transaction{},
	); err != nil {
		logger.Panic("cannot migrate db", zap.Error(err))
	}

	q := queue.NewQueueService()
	commandSvc := command.NewSlackService(logger, db, slackClient)
	userSvc := user.NewPGService(db)
	itemSvc := item.NewPGService(logger, notionClient, db)
	eventSvc := event.NewSlackService(logger, db, slackClient, notionClient)

	c := cron.New()
	c.AddFunc("@every 0h5m00s", func() {
		// this will avoid multiple cronjobs may read same data at sometime
		logger.Info("start handling msg")
		if !q.GetIsConsuming() {
			q.SetIsConsuming(true)
			var next interface{}

			for e := q.Consume(); e != nil; e = next {
				if u, ok := e.(*model.User); ok {
					sUser, err := slackClient.GetUserInfo(u.ID)
					if err != nil {
						logger.Error("cannot get slack user info", zap.Error(err), zap.String("user_id", u.ID))
						continue
					}
					if sUser.IsBot {
						continue
					}
					_, _, err = userSvc.Update(u.ID, map[string]interface{}{
						"exp":            u.Exp,
						"first_name":     sUser.Profile.FirstName,
						"last_name":      sUser.Profile.LastName,
						"real_name":      sUser.Profile.RealName,
						"display_name":   sUser.Profile.DisplayName,
						"tz":             sUser.TZ,
						"image_original": sUser.Profile.ImageOriginal,
						"slack_email":    sUser.Profile.Email,
					})
					if err != nil {
						logger.Error("cannot update user exp", zap.Error(err), zap.String("user_id", u.ID))
					}
					// if isLevelUp && u.SlackChannel != "" {
					// 	slackClient.PostMessage(u.SlackChannel, slack.MsgOptionText(fmt.Sprintf("User %v is level up!", u.ID), false))
					// }
				}
				next = q.Consume()
				time.Sleep(100 * time.Millisecond)
			}

			q.SetIsConsuming(false)
		}
		logger.Info("end handling msg")
	})

	c.AddFunc("@every 0h1m00s", func() {
		logger.Info("start syncing redeem")
		isExpired := false
		items, err := notionClient.QueryDatabase(context.Background(), os.Getenv("NOTION_DATABASE_ID"), &notion.DatabaseQuery{Filter: &notion.DatabaseQueryFilter{
			Property: "Expired",
			Checkbox: &notion.CheckboxDatabaseQueryFilter{Equals: &isExpired},
		}})
		if err != nil {
			logger.Error("cannot fetch items", zap.Error(err))
			return
		}

		for _, v := range items.Results {
			var count int64
			if err := db.Model(&model.Transaction{}).Where("item_id = ?", v.ID).Count(&count).Error; err != nil {
				logger.Error("cannot count transaction", zap.Error(err), zap.String("item_id", v.ID))
				continue
			}
			if count == 0 {
				continue
			}

			redeemed := float64(count)
			notionClient.UpdatePageProps(context.Background(),
				v.ID,
				notion.UpdatePageParams{DatabasePageProperties: &notion.DatabasePageProperties{
					"Redeemed": notion.DatabasePageProperty{
						Type:   "number",
						Number: &redeemed,
					}}},
			)
		}
		logger.Info("end syncing redeem")
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
	handler.NewInteractiveHandler(e, logger, q, userSvc, itemSvc)
	handler.NewAuthorizeHandler(e, logger, db, slackClient)

	e.Logger.Fatal(e.Start(":8080"))
}
