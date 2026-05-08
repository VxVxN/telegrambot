package bot

import (
	"strings"

	"gopkg.in/telebot.v3"
)

func (b *Bot) handleText(c telebot.Context) error {
	text := strings.ToLower(c.Text())
	userID := int(c.Sender().ID)

	switch {
	case text == "хелп" || text == "help":
		return b.handleHelp(c)

	case text == "список" || text == "list":
		return b.handleTodayTodos(c, userID)

	case text == "полный список" || text == "full list":
		return b.handleAllTodos(c, userID)

	case strings.HasPrefix(text, "добавить ") || strings.HasPrefix(text, "add "):
		return b.handleAddTodo(c, userID)

	case strings.HasPrefix(text, "удалить ") || strings.HasPrefix(text, "delete "):
		return b.handleDeleteTodo(c, userID)

	case text == "очистить" || text == "clear":
		return b.handleClearTodos(c, userID)
		
	case text == "цены" || text == "prices":
		return b.handleAllPrices(c)
	case text == "usd" || text == "доллар" || text == "курс доллара":
		return b.handleUSDRate(c)
	case text == "подписаться" || text == "subscribe":
		return b.handleSubscribe(c)
	case text == "отписаться" || text == "unsubscribe":
		return b.handleUnsubscribe(c)
	}
	return nil
}
