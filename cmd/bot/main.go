package main

import (
	"log"
	"os"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/VxVxN/telegrambot/internal/bot"
	"github.com/VxVxN/telegrambot/internal/currency"
	"github.com/VxVxN/telegrambot/internal/news"
	"github.com/VxVxN/telegrambot/internal/storage"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable not set")
	}

	tb, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	todos := storage.NewTodoStore("todos.json")
	subs := storage.NewSubscriberStore("subscribers.json")
	notifications := storage.NewNotificationStore("notification.json")
	newsState := storage.NewNewsStore("news.json")
	cur := currency.NewClient()
	newsClient := news.NewClient()

	b := bot.New(tb, todos, subs, notifications, newsState, cur, newsClient)
	b.RegisterHandlers()

	log.Println("Bot is running")
	tb.Start()
}
