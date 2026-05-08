package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

func (b *Bot) handleAllPrices(c telebot.Context) error {
	btcPrice, err := b.currency.FetchCryptoPrice("bitcoin")
	if err != nil {
		return c.Send("Failed to get prices")
	}
	ethPrice, err := b.currency.FetchCryptoPrice("ethereum")
	if err != nil {
		return c.Send("Failed to get prices")
	}
	xrpPrice, err := b.currency.FetchCryptoPrice("ripple")
	if err != nil {
		return c.Send("Failed to get prices")
	}

	msg := fmt.Sprintf("Current Prices:\n\nBTC: $%.2f USD\nETH: $%.2f USD\nXRP: $%.2f USD",
		btcPrice, ethPrice, xrpPrice)

	if usdRate, err := b.currency.FetchUSDRate(); err == nil {
		msg += fmt.Sprintf("\nUSD/RUB: %.2f RUB", usdRate)
	}

	msg += fmt.Sprintf("\n\nUpdated: %s", time.Now().Format("15:04:05"))
	return c.Send(msg)
}

func (b *Bot) handleUSDRate(c telebot.Context) error {
	rate, err := b.currency.FetchUSDRate()
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to get USD rate: %v", err))
	}
	return c.Send(fmt.Sprintf("USD to RUB exchange rate:\n1 USD = %.2f RUB\n\nUpdated: %s",
		rate, time.Now().Format("02.01.2006 15:04:05")))
}

func (b *Bot) handleSubscribe(c telebot.Context) error {
	if !b.subs.Subscribe(c.Sender().ID) {
		return c.Send("You are already subscribed to daily price notifications at 10:00 AM")
	}
	return c.Send("You have subscribed to daily price notifications at 10:00 AM!\nUse /unsubscribe to stop.")
}

func (b *Bot) handleUnsubscribe(c telebot.Context) error {
	if !b.subs.Unsubscribe(c.Sender().ID) {
		return c.Send("You are not subscribed to price notifications")
	}
	return c.Send("You have unsubscribed from daily price notifications")
}

func (b *Bot) runDailyPriceNotifications() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))
		b.sendDailyPriceUpdates()
	}
}

func (b *Bot) sendDailyPriceUpdates() {
	ids := b.subs.List()
	if len(ids) == 0 {
		return
	}

	btcPrice, err := b.currency.FetchCryptoPrice("bitcoin")
	if err != nil {
		log.Printf("Error fetching BTC price: %v", err)
		return
	}
	ethPrice, err := b.currency.FetchCryptoPrice("ethereum")
	if err != nil {
		log.Printf("Error fetching ETH price: %v", err)
		return
	}
	xrpPrice, err := b.currency.FetchCryptoPrice("ripple")
	if err != nil {
		log.Printf("Error fetching XRP price: %v", err)
		return
	}

	msg := fmt.Sprintf("Good morning! Today's prices:\n\nBTC: $%.2f\nETH: $%.2f\nXRP: $%.2f",
		btcPrice, ethPrice, xrpPrice)

	if usdRate, err := b.currency.FetchUSDRate(); err == nil {
		msg += fmt.Sprintf("\nUSD/RUB: %.2f RUB", usdRate)
	}
	msg += "\n\nUse /prices for latest prices anytime!"

	for _, userID := range ids {
		user := &telebot.User{ID: userID}
		if _, err := b.tb.Send(user, msg); err != nil {
			log.Printf("Error sending notification to user %d: %v", userID, err)
			if strings.Contains(err.Error(), "bot was blocked") {
				b.subs.Remove(userID)
			}
		}
	}
}
