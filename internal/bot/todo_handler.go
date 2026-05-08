package bot

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VxVxN/telegrambot/internal/model"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleHelp(c telebot.Context) error {
	data, err := os.ReadFile("help.txt")
	if err != nil {
		return c.Send("Failed to load help")
	}
	return c.Send(string(data))
}

func (b *Bot) handleTodayTodos(c telebot.Context, userID int) error {
	todos := b.todos.TodayTodos(userID)
	if len(todos) == 0 {
		return c.Send("No tasks for today")
	}

	var msg strings.Builder
	msg.WriteString("Your todo list for today:\n\n")
	for _, t := range todos {
		writeTodoEntry(&msg, t)
	}
	return c.Send(msg.String())
}

func (b *Bot) handleAllTodos(c telebot.Context, userID int) error {
	todos := b.todos.AllTodos(userID)
	if len(todos) == 0 {
		return c.Send("Your todo list is empty")
	}

	var msg strings.Builder
	msg.WriteString("Your full todo list:\n\n")
	for _, t := range todos {
		writeTodoEntry(&msg, t)
	}
	return c.Send(msg.String())
}

func (b *Bot) handleAddTodo(c telebot.Context, userID int) error {
	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: add [date?] [text]\nExamples:\n- add Buy milk\n- add 25.12.2023 Buy gifts")
	}

	var date time.Time
	var todoText string

	if d, err := time.Parse("02.01.2006", parts[1]); err == nil {
		if len(parts) < 3 {
			return c.Send("Please provide task text after the date")
		}
		date = d
		todoText = strings.Join(parts[2:], " ")
	} else {
		date = time.Now()
		todoText = strings.Join(parts[1:], " ")
	}

	todo := b.todos.Add(userID, todoText, date)

	return c.Send(fmt.Sprintf("ID %d: %s\nTask added!\nDate: %s\n\nTo make it repeating, use:\n/repeat_daily %d\n/repeat_weekly %d [days]\n/repeat_custom %d [days]",
		todo.ID, todo.Text, date.Format("02.01.2006"), todo.ID, todo.ID, todo.ID))
}

func (b *Bot) handleDeleteTodo(c telebot.Context, userID int) error {
	parts := strings.Fields(c.Text())
	if len(parts) != 2 {
		return c.Send("Usage: delete [ID]\nExample: delete 1")
	}

	var id int
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return c.Send("Invalid task ID")
	}

	if b.todos.Delete(userID, id) {
		return c.Send("Task deleted")
	}
	return c.Send("Task with specified ID not found")
}

func (b *Bot) handleClearTodos(c telebot.Context, userID int) error {
	b.todos.Clear(userID)
	return c.Send("All tasks deleted")
}

func (b *Bot) handleRepeatDaily(c telebot.Context) error {
	parts := strings.Fields(c.Text())
	if len(parts) != 2 {
		return c.Send("Usage: /repeat_daily [ID]")
	}

	var id int
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return c.Send("Invalid task ID")
	}

	todo, ok := b.todos.SetRepeat(int(c.Sender().ID), id, "daily", 0, nil)
	if !ok {
		return c.Send("Task with specified ID not found")
	}
	return c.Send("Task is now repeating: " + todo.RepeatText())
}

func (b *Bot) handleRepeatWeekly(c telebot.Context) error {
	parts := strings.Fields(c.Text())
	if len(parts) < 3 {
		return c.Send("Usage: /repeat_weekly [ID] [week days]\nExample: /repeat_weekly 1 monday wednesday friday")
	}

	var id int
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return c.Send("Invalid task ID")
	}

	days := parts[2:]
	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true,
		"thursday": true, "friday": true, "saturday": true, "sunday": true,
	}
	for _, day := range days {
		if !validDays[strings.ToLower(day)] {
			return c.Send("Invalid day of week. Use: monday, tuesday, wednesday, thursday, friday, saturday, sunday")
		}
	}

	todo, ok := b.todos.SetRepeat(int(c.Sender().ID), id, "weekly", 0, days)
	if !ok {
		return c.Send("Task with specified ID not found")
	}
	return c.Send("Task is now repeating: " + todo.RepeatText())
}

func (b *Bot) handleRepeatCustom(c telebot.Context) error {
	parts := strings.Fields(c.Text())
	if len(parts) != 3 {
		return c.Send("Usage: /repeat_custom [ID] [interval in days]\nExample: /repeat_custom 1 5")
	}

	var id, interval int
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return c.Send("Invalid task ID")
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &interval); err != nil || interval <= 0 {
		return c.Send("Interval must be a positive number")
	}

	todo, ok := b.todos.SetRepeat(int(c.Sender().ID), id, "custom", interval, nil)
	if !ok {
		return c.Send("Task with specified ID not found")
	}
	return c.Send("Task is now repeating: " + todo.RepeatText())
}

func writeTodoEntry(msg *strings.Builder, todo model.Todo) {
	fmt.Fprintf(msg, "ID %d: %s\nDate: %s\n", todo.ID, todo.Text, todo.Date.Format("02.01.2006"))
	if todo.Repeat != "none" {
		fmt.Fprintf(msg, "Repeat: %s\n", todo.RepeatText())
	}
	msg.WriteString("---\n")
}
