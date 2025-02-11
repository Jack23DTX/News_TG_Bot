package main

import (
	"TgNewsPet/internal/bot"
	"TgNewsPet/internal/bot/middleware"
	"TgNewsPet/internal/botkit"
	"TgNewsPet/internal/config"
	"TgNewsPet/internal/fetcher"
	"TgNewsPet/internal/notifier"
	"TgNewsPet/internal/storage"
	"TgNewsPet/internal/summary"
	"context"
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	botAPI, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		log.Printf("failed to create bot: %v", err)
		return
	}

	db, err := sqlx.Connect("postgres", config.Get().DatabaseDSN)
	if err != nil {
		log.Printf("failed to connect to database: %v", err)
		return
	}
	defer db.Close()

	var (
		articleStorage = storage.NewArticleStorage(db)
		sourceStorage  = storage.NewSourceStorage(db)
		fetcher        = fetcher.New(
			articleStorage,
			sourceStorage,
			config.Get().FetchInterval,
			config.Get().FilterKeywords,
		)
		notifier = notifier.New(
			articleStorage,
			summary.NewOpenAISummarizer(config.Get().OpenAIKey, config.Get().OpenaiModel, config.Get().OpenAIPrompt),
			botAPI,
			config.Get().NotificationInterval,
			2*config.Get().FetchInterval,
			config.Get().TelegramChannelID,
		)
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	newsBot := botkit.New(botAPI)
	newsBot.RegisterCmdView("start", bot.ViewCmdStart())
	newsBot.RegisterCmdView(
		"addsource",
		middleware.AdminOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdAddSource(sourceStorage),
		),
	)

	newsBot.RegisterCmdView(
		"listsources",
		middleware.AdminOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdListSources(sourceStorage),
		),
	)

	newsBot.RegisterCmdView(
		"deletesource",
		middleware.AdminOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdDeleteSource(sourceStorage),
		),
	)

	go func(ctx context.Context) {
		if err := fetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run fetcher: %v", err)
				return
			}

			log.Printf("[INFO] fetcher stopped")
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := notifier.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run notifier: %v", err)
				return
			}
			log.Printf("[INFO] notifier stopped")
		}
	}(ctx)

	if err := newsBot.Run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Printf("[ERROR] failed to start bot: %v", err)
			return
		}

		log.Println("[ERROR] bot stopped")
	}
}
