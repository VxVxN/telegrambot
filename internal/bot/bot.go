package bot

import (
	"gopkg.in/telebot.v3"

	"github.com/VxVxN/telegrambot/internal/currency"
	"github.com/VxVxN/telegrambot/internal/storage"
)

type Bot struct {
	tb       *telebot.Bot
	todos    *storage.TodoStore
	subs     *storage.SubscriberStore
	currency *currency.Client
}

func New(tb *telebot.Bot, todos *storage.TodoStore, subs *storage.SubscriberStore, cur *currency.Client) *Bot {
	return &Bot{
		tb:       tb,
		todos:    todos,
		subs:     subs,
		currency: cur,
	}
}

func (b *Bot) RegisterHandlers() {
	b.tb.Handle(telebot.OnText, b.handleText)

	b.tb.Handle("/repeat_daily", b.handleRepeatDaily)
	b.tb.Handle("/repeat_weekly", b.handleRepeatWeekly)
	b.tb.Handle("/repeat_custom", b.handleRepeatCustom)

	b.tb.Handle("/prices", b.handleAllPrices)
	b.tb.Handle("/usd", b.handleUSDRate)
	b.tb.Handle("/subscribe", b.handleSubscribe)
	b.tb.Handle("/unsubscribe", b.handleUnsubscribe)

	go b.runDailyPriceNotifications()
}
