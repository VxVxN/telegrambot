package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

type Todo struct {
	ID       int       `json:"id"`
	Text     string    `json:"text"`
	Date     time.Time `json:"date"`
	Repeat   string    `json:"repeat"`
	Interval int       `json:"interval"`
	Days     []string  `json:"days"`
}

type UserTodos struct {
	UserID int    `json:"user_id"`
	Todos  []Todo `json:"todos"`
}

var todosData = make(map[int]*UserTodos)
var nextID = 1

var priceSubscribers = make(map[int64]bool)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable not set")
	}

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	loadTodos()
	loadSubscribers()

	go startDailyPriceNotifications(bot)

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		text := strings.ToLower(c.Text())
		userID := int(c.Sender().ID)

		switch {
		case text == "хелп" || text == "help":
			data, err := os.ReadFile("help.txt")
			if err != nil {
				return err
			}
			return c.Send(string(data))

		case text == "список" || text == "list":
			return showTodayTodos(c, userID)

		case text == "полный список" || text == "full list":
			return showAllTodos(c, userID)

		case strings.HasPrefix(text, "добавить ") || strings.HasPrefix(text, "add "):
			return addTodo(c, userID, text)

		case strings.HasPrefix(text, "удалить ") || strings.HasPrefix(text, "delete "):
			return deleteTodo(c, userID, text)

		case text == "очистить" || text == "clear":
			return clearTodos(c, userID)

		case text == "eth":
			return getPrice(c, "ethereum")
		case text == "btc":
			return getPrice(c, "bitcoin")
		case text == "xrp":
			return getPrice(c, "ripple")
		case text == "цены" || text == "prices":
			return getAllPrices(c)
		case text == "подписаться" || text == "subscribe":
			return subscribeToPrices(c)
		case text == "отписаться" || text == "unsubscribe":
			return unsubscribeFromPrices(c)
		}
		return nil
	})

	bot.Handle("/repeat_daily", repeatDaily)
	bot.Handle("/repeat_weekly", repeatWeekly)
	bot.Handle("/repeat_custom", repeatCustom)

	bot.Handle("/prices", getAllPrices)
	bot.Handle("/subscribe", subscribeToPrices)
	bot.Handle("/unsubscribe", unsubscribeFromPrices)

	log.Println("Bot is running")
	bot.Start()
}

func getPrice(c telebot.Context, cryptoID string) error {
	price, err := fetchCryptoPrice(cryptoID)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Failed to get %s price: %v", strings.ToUpper(cryptoID), err))
	}

	var symbol string
	switch cryptoID {
	case "ethereum":
		symbol = "ETH"
	case "bitcoin":
		symbol = "BTC"
	case "ripple":
		symbol = "XRP"
	}

	return c.Send(fmt.Sprintf("💰 %s price: $%.2f USD", symbol, price))
}

func getAllPrices(c telebot.Context) error {
	ethPrice, err := fetchCryptoPrice("ethereum")
	if err != nil {
		return c.Send("❌ Failed to get prices")
	}

	btcPrice, err := fetchCryptoPrice("bitcoin")
	if err != nil {
		return c.Send("❌ Failed to get prices")
	}

	xrpPrice, err := fetchCryptoPrice("ripple")
	if err != nil {
		return c.Send("❌ Failed to get prices")
	}

	message := fmt.Sprintf("📊 Current Cryptocurrency Prices:\n\n💰 BTC: $%v\n💰 ETH: $%v\n💰 XRP: $%v\n\n🕐 Updated: %s",
		btcPrice, ethPrice, xrpPrice, time.Now().Format("15:04:05"))

	return c.Send(message)
}

func subscribeToPrices(c telebot.Context) error {
	userID := c.Sender().ID
	if priceSubscribers[userID] {
		return c.Send("✅ You are already subscribed to daily price notifications at 10:00 AM")
	}

	priceSubscribers[userID] = true
	saveSubscribers()
	return c.Send("✅ You have subscribed to daily price notifications at 10:00 AM!\nUse /unsubscribe to stop receiving notifications.")
}

func unsubscribeFromPrices(c telebot.Context) error {
	userID := c.Sender().ID
	if !priceSubscribers[userID] {
		return c.Send("❌ You are not subscribed to price notifications")
	}

	delete(priceSubscribers, userID)
	saveSubscribers()
	return c.Send("✅ You have unsubscribed from daily price notifications")
}

func startDailyPriceNotifications(bot *telebot.Bot) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())

		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		waitDuration := next.Sub(now)
		time.Sleep(waitDuration)

		sendDailyPriceUpdates(bot)
	}
}

func sendDailyPriceUpdates(bot *telebot.Bot) {
	if len(priceSubscribers) == 0 {
		return
	}

	ethPrice, err := fetchCryptoPrice("ethereum")
	if err != nil {
		log.Printf("Error fetching ETH price: %v", err)
		return
	}

	btcPrice, err := fetchCryptoPrice("bitcoin")
	if err != nil {
		log.Printf("Error fetching BTC price: %v", err)
		return
	}

	xrpPrice, err := fetchCryptoPrice("ripple")
	if err != nil {
		log.Printf("Error fetching XRP price: %v", err)
		return
	}

	message := fmt.Sprintf("🌅 Good morning! Here are today's cryptocurrency prices at 10:00 AM:\n\n💰 BTC: $%v\n💰 ETH: $%v\n💰 XRP: $%v\n\nUse /prices to get the latest prices anytime!",
		btcPrice, ethPrice, xrpPrice)

	for userID := range priceSubscribers {
		user := &telebot.User{ID: userID}
		_, err := bot.Send(user, message)
		if err != nil {
			log.Printf("Error sending price notification to user %d: %v", userID, err)
			if strings.Contains(err.Error(), "bot was blocked") {
				delete(priceSubscribers, userID)
				saveSubscribers()
			}
		}
	}
}

func fetchCryptoPrice(cryptoID string) (float64, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", cryptoID)

	resp, err := httpGet(url)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return 0, err
	}

	if priceData, ok := result[cryptoID].(map[string]interface{}); ok {
		if price, ok := priceData["usd"].(float64); ok {
			return price, nil
		}
	}

	return 0, fmt.Errorf("price not found")
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func saveSubscribers() {
	data, err := json.Marshal(priceSubscribers)
	if err != nil {
		log.Printf("Error saving subscribers: %v", err)
		return
	}

	err = os.WriteFile("subscribers.json", data, 0644)
	if err != nil {
		log.Printf("Error writing subscribers file: %v", err)
	}
}

func loadSubscribers() {
	data, err := os.ReadFile("subscribers.json")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Error reading subscribers file: %v", err)
		return
	}

	err = json.Unmarshal(data, &priceSubscribers)
	if err != nil {
		log.Printf("Error parsing subscribers data: %v", err)
	}
}

func showTodayTodos(c telebot.Context, userID int) error {
	userTodos, exists := todosData[userID]
	if !exists || len(userTodos.Todos) == 0 {
		return c.Send("📝 Your todo list is empty")
	}

	today := time.Now().Format("02.01.2006")
	var message strings.Builder
	message.WriteString("📋 Your todo list for today:\n\n")

	hasTasks := false
	for _, todo := range userTodos.Todos {
		todoDate := todo.Date.Format("02.01.2006")

		if todoDate == today || isRepeatingTaskDueToday(todo) {
			hasTasks = true
			message.WriteString(fmt.Sprintf("🆔 %d: %s\n", todo.ID, todo.Text))
			message.WriteString(fmt.Sprintf("📅 Date: %s\n", todoDate))

			if todo.Repeat != "none" {
				repeatText := getRepeatText(todo)
				message.WriteString(fmt.Sprintf("🔄 Repeat: %s\n", repeatText))
			}
			message.WriteString("---\n")
		}
	}

	if !hasTasks {
		return c.Send("📝 No tasks for today")
	}

	return c.Send(message.String())
}

func showAllTodos(c telebot.Context, userID int) error {
	userTodos, exists := todosData[userID]
	if !exists || len(userTodos.Todos) == 0 {
		return c.Send("📝 Your todo list is empty")
	}

	var message strings.Builder
	message.WriteString("📋 Your full todo list:\n\n")

	for _, todo := range userTodos.Todos {
		message.WriteString(fmt.Sprintf("🆔 %d: %s\n", todo.ID, todo.Text))
		message.WriteString(fmt.Sprintf("📅 Date: %s\n", todo.Date.Format("02.01.2006")))

		if todo.Repeat != "none" {
			repeatText := getRepeatText(todo)
			message.WriteString(fmt.Sprintf("🔄 Repeat: %s\n", repeatText))
		}
		message.WriteString("---\n")
	}

	return c.Send(message.String())
}

func addTodo(c telebot.Context, userID int, text string) error {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return c.Send("❌ Usage: add [date?] [text]\nExamples:\n• add Buy milk\n• add 25.12.2023 Buy gifts")
	}

	var date time.Time
	var todoText string
	var err error

	date, err = time.Parse("02.01.2006", parts[1])
	if err == nil {
		if len(parts) < 3 {
			return c.Send("❌ Please provide task text after the date")
		}
		todoText = strings.Join(parts[2:], " ")
	} else {
		date = time.Now()
		todoText = strings.Join(parts[1:], " ")
	}

	todo := Todo{
		ID:     nextID,
		Text:   todoText,
		Date:   date,
		Repeat: "none",
	}
	nextID++

	if _, exists := todosData[userID]; !exists {
		todosData[userID] = &UserTodos{
			UserID: userID,
			Todos:  []Todo{},
		}
	}
	todosData[userID].Todos = append(todosData[userID].Todos, todo)

	saveTodos()

	return c.Send(fmt.Sprintf("🆔 %d: %s\n✅ Task added!\n📅 Date: %s\n\nTo make it repeating, use commands:\n• /repeat_daily [ID] - repeat daily\n• /repeat_weekly [ID] [days] - repeat on specific week days\n• /repeat_custom [ID] [days] - repeat every N days",
		todo.ID, todo.Text, date.Format("02.01.2006")))
}

func deleteTodo(c telebot.Context, userID int, text string) error {
	parts := strings.Fields(text)
	if len(parts) != 2 {
		return c.Send("❌ Usage: delete [ID]\nExample: delete 1")
	}

	var id int
	_, err := fmt.Sscanf(parts[1], "%d", &id)
	if err != nil {
		return c.Send("❌ Invalid task ID")
	}

	userTodos, exists := todosData[userID]
	if !exists {
		return c.Send("❌ You don't have any tasks")
	}

	for i, todo := range userTodos.Todos {
		if todo.ID == id {
			userTodos.Todos = append(userTodos.Todos[:i], userTodos.Todos[i+1:]...)
			saveTodos()
			return c.Send("✅ Task deleted")
		}
	}

	return c.Send("❌ Task with specified ID not found")
}

func clearTodos(c telebot.Context, userID int) error {
	if _, exists := todosData[userID]; exists {
		todosData[userID].Todos = []Todo{}
		saveTodos()
	}
	return c.Send("✅ All tasks deleted")
}

func repeatDaily(c telebot.Context) error {
	userID := int(c.Sender().ID)
	parts := strings.Fields(c.Text())

	if len(parts) != 2 {
		return c.Send("❌ Usage: /repeat_daily [ID]")
	}

	var id int
	_, err := fmt.Sscanf(parts[1], "%d", &id)
	if err != nil {
		return c.Send("❌ Invalid task ID")
	}

	return updateTodoRepeat(userID, id, "daily", 0, nil, c)
}

func repeatWeekly(c telebot.Context) error {
	userID := int(c.Sender().ID)
	parts := strings.Fields(c.Text())

	if len(parts) < 3 {
		return c.Send("❌ Usage: /repeat_weekly [ID] [week days]\nExample: /repeat_weekly 1 monday wednesday friday")
	}

	var id int
	_, err := fmt.Sscanf(parts[1], "%d", &id)
	if err != nil {
		return c.Send("❌ Invalid task ID")
	}

	days := parts[2:]
	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true,
		"thursday": true, "friday": true, "saturday": true, "sunday": true,
	}

	for _, day := range days {
		if !validDays[strings.ToLower(day)] {
			return c.Send("❌ Invalid day of week. Use: monday, tuesday, wednesday, thursday, friday, saturday, sunday")
		}
	}

	return updateTodoRepeat(userID, id, "weekly", 0, days, c)
}

func repeatCustom(c telebot.Context) error {
	userID := int(c.Sender().ID)
	parts := strings.Fields(c.Text())

	if len(parts) != 3 {
		return c.Send("❌ Usage: /repeat_custom [ID] [interval in days]\nExample: /repeat_custom 1 5")
	}

	var id, interval int
	_, err := fmt.Sscanf(parts[1], "%d", &id)
	if err != nil {
		return c.Send("❌ Invalid task ID")
	}
	_, err = fmt.Sscanf(parts[2], "%d", &interval)
	if err != nil || interval <= 0 {
		return c.Send("❌ Interval must be a positive number")
	}

	return updateTodoRepeat(userID, id, "custom", interval, nil, c)
}

func updateTodoRepeat(userID, id int, repeatType string, interval int, days []string, c telebot.Context) error {
	userTodos, exists := todosData[userID]
	if !exists {
		return c.Send("❌ You don't have any tasks")
	}

	for i := range userTodos.Todos {
		if userTodos.Todos[i].ID == id {
			userTodos.Todos[i].Repeat = repeatType
			userTodos.Todos[i].Interval = interval
			userTodos.Todos[i].Days = days
			saveTodos()

			repeatText := getRepeatText(userTodos.Todos[i])
			return c.Send("✅ Task is now repeating: " + repeatText)
		}
	}

	return c.Send("❌ Task with specified ID not found")
}

func getRepeatText(todo Todo) string {
	switch todo.Repeat {
	case "daily":
		return "daily"
	case "weekly":
		return "on " + strings.Join(todo.Days, ", ")
	case "custom":
		return fmt.Sprintf("every %d days", todo.Interval)
	default:
		return "no repeat"
	}
}

func isRepeatingTaskDueToday(todo Todo) bool {
	today := time.Now()
	todayWeekday := strings.ToLower(today.Weekday().String())

	switch todo.Repeat {
	case "daily":
		return true
	case "weekly":
		for _, day := range todo.Days {
			if strings.ToLower(day) == todayWeekday {
				return true
			}
		}
	case "custom":
		daysDiff := int(today.Sub(todo.Date).Hours() / 24)
		return daysDiff >= 0 && daysDiff%todo.Interval == 0
	}
	return false
}

func saveTodos() {
	data, err := json.Marshal(todosData)
	if err != nil {
		log.Printf("Error saving data: %v", err)
		return
	}

	err = os.WriteFile("todos.json", data, 0644)
	if err != nil {
		log.Printf("Error writing file: %v", err)
	}
}

func loadTodos() {
	data, err := os.ReadFile("todos.json")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Error reading file: %v", err)
		return
	}

	err = json.Unmarshal(data, &todosData)
	if err != nil {
		log.Printf("Error parsing data: %v", err)
	}
}
