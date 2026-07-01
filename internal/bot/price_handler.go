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
		return c.Send("You are already subscribed to daily prices (10:00 AM) and Go blog news")
	}
	return c.Send("You have subscribed to daily prices at 10:00 AM and new Go blog posts!\nUse /unsubscribe to stop.")
}

func (b *Bot) handleUnsubscribe(c telebot.Context) error {
	if !b.subs.Unsubscribe(c.Sender().ID) {
		return c.Send("You are not subscribed to notifications")
	}
	return c.Send("You have unsubscribed from daily prices and Go blog news")
}

const dailyNotificationHour = 10

// runDailyPriceNotifications wakes up once per minute and checks the wall-clock
// time instead of sleeping until a single precomputed moment. A long
// time.Sleep is driven by the monotonic clock, so it pauses while the host is
// suspended and overshoots the target by the suspend duration — which made
// notifications arrive at random times or get skipped entirely.
//
// On each tick it sends today's notification if the scheduled time has already
// passed and it hasn't been sent today yet. This is a catch-up: if the host was
// off or asleep at 10:00 and comes back at, say, 11:30, the very first check
// (which also runs immediately on startup) delivers the missed message. The
// last successful date is persisted, so a mid-day restart never re-sends. On a
// send error the date is not recorded, so the next tick retries until it works.
func (b *Bot) runDailyPriceNotifications() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		now := time.Now()
		today := now.Format("2006-01-02")
		scheduled := time.Date(now.Year(), now.Month(), now.Day(), dailyNotificationHour, 0, 0, 0, now.Location())

		if !now.Before(scheduled) && b.notifications.LastSent() != today {
			if err := b.sendDailyPriceUpdates(); err != nil {
				log.Printf("Daily notification failed, will retry next minute: %v", err)
			} else {
				b.notifications.SetLastSent(today)
			}
		}
		<-ticker.C
	}
}

func (b *Bot) sendDailyPriceUpdates() error {
	ids := b.subs.List()
	if len(ids) == 0 {
		return nil
	}

	btcPrice, err := b.currency.FetchCryptoPrice("bitcoin")
	if err != nil {
		return fmt.Errorf("fetch BTC price: %w", err)
	}
	ethPrice, err := b.currency.FetchCryptoPrice("ethereum")
	if err != nil {
		return fmt.Errorf("fetch ETH price: %w", err)
	}
	xrpPrice, err := b.currency.FetchCryptoPrice("ripple")
	if err != nil {
		return fmt.Errorf("fetch XRP price: %w", err)
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
	return nil
}
