package main

import (
	"encoding/json"
	"fmt"
	"log"
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

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		text := strings.ToLower(c.Text())
		userID := int(c.Sender().ID)

		switch {
		case text == "хелп", text == "help":
			data, err := os.ReadFile("help.txt")
			if err != nil {
				return err
			}
			return c.Send(string(data))

		case text == "список", text == "list":
			return showTodayTodos(c, userID)

		case text == "полный список", text == "full list":
			return showAllTodos(c, userID)

		case strings.HasPrefix(text, "добавить ") || strings.HasPrefix(text, "add "):
			return addTodo(c, userID, text)

		case strings.HasPrefix(text, "удалить ") || strings.HasPrefix(text, "delete "):
			return deleteTodo(c, userID, text)

		case text == "очистить", text == "clear":
			return clearTodos(c, userID)
		}
		return nil
	})

	bot.Handle("/repeat_daily", repeatDaily)
	bot.Handle("/repeat_weekly", repeatWeekly)
	bot.Handle("/repeat_custom", repeatCustom)

	log.Println("Bot is running")
	bot.Start()
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
