package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/VxVxN/telegrambot/internal/news"
)

const (
	// newsPollInterval is how often the feed is polled for new posts.
	newsPollInterval = 1 * time.Hour
	// newsOnDemandLimit is how many recent posts /news shows.
	newsOnDemandLimit = 5
)

// handleNews replies with the latest posts from the Go blog on demand.
func (b *Bot) handleNews(c telebot.Context) error {
	items, err := b.news.FetchLatest(newsOnDemandLimit)
	if err != nil {
		return c.Send("Failed to get news")
	}
	if len(items) == 0 {
		return c.Send("No news available")
	}
	return c.Send(formatNewsList(items), telebot.NoPreview)
}

func formatNewsList(items []news.Item) string {
	var sb strings.Builder
	sb.WriteString("📰 Latest from the Go blog\n")
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("\n• %s", it.Title))
		if d := formatNewsDate(it.Published); d != "" {
			sb.WriteString(fmt.Sprintf("\n  🗓 %s", d))
		}
		sb.WriteString(fmt.Sprintf("\n  %s\n", it.URL))
	}
	return sb.String()
}

func formatNewsItem(it news.Item) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📰 New Go blog post\n\n%s", it.Title))

	meta := make([]string, 0, 2)
	if d := formatNewsDate(it.Published); d != "" {
		meta = append(meta, "🗓 "+d)
	}
	if it.Author != "" {
		meta = append(meta, "✍ "+it.Author)
	}
	if len(meta) > 0 {
		sb.WriteString("\n" + strings.Join(meta, "  ·  "))
	}

	if it.Summary != "" {
		sb.WriteString(fmt.Sprintf("\n\n%s", it.Summary))
	}
	sb.WriteString(fmt.Sprintf("\n\n%s", it.URL))
	return sb.String()
}

// formatNewsDate renders a publication date as "02 Jan 2006", or "" if unknown.
func formatNewsDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02 Jan 2006")
}

// runNewsNotifications polls the feed on a fixed interval and broadcasts any
// posts published since the last one already sent. It runs an immediate check
// on startup. On the very first run ever (no stored URL) it records the newest
// post without sending, so subscribers are not flooded with the whole backlog.
func (b *Bot) runNewsNotifications() {
	ticker := time.NewTicker(newsPollInterval)
	defer ticker.Stop()

	for {
		if err := b.broadcastNewNews(); err != nil {
			log.Printf("News poll failed, will retry next interval: %v", err)
		}
		<-ticker.C
	}
}

func (b *Bot) broadcastNewNews() error {
	items, err := b.news.FetchLatest(0)
	if err != nil {
		return fmt.Errorf("fetch feed: %w", err)
	}
	if len(items) == 0 {
		return nil
	}

	newest := items[0].URL
	lastURL := b.newsState.LastURL()

	// First run: remember where we are, don't replay history.
	if lastURL == "" {
		b.newsState.SetLastURL(newest)
		return nil
	}
	if newest == lastURL {
		return nil
	}

	// Collect items newer than the last sent one (feed is newest-first).
	var fresh []news.Item
	for _, it := range items {
		if it.URL == lastURL {
			break
		}
		fresh = append(fresh, it)
	}

	// Send oldest-first so the timeline reads naturally.
	for i := len(fresh) - 1; i >= 0; i-- {
		b.sendNewsToSubscribers(formatNewsItem(fresh[i]))
	}

	b.newsState.SetLastURL(newest)
	return nil
}

func (b *Bot) sendNewsToSubscribers(msg string) {
	for _, userID := range b.subs.List() {
		user := &telebot.User{ID: userID}
		if _, err := b.tb.Send(user, msg, telebot.NoPreview); err != nil {
			log.Printf("Error sending news to user %d: %v", userID, err)
			if strings.Contains(err.Error(), "bot was blocked") {
				b.subs.Remove(userID)
			}
		}
	}
}
