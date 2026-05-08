package model

import (
	"fmt"
	"strings"
	"time"
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

func (t Todo) RepeatText() string {
	switch t.Repeat {
	case "daily":
		return "daily"
	case "weekly":
		return "on " + strings.Join(t.Days, ", ")
	case "custom":
		return fmt.Sprintf("every %d days", t.Interval)
	default:
		return "no repeat"
	}
}

func (t Todo) IsDueToday() bool {
	today := time.Now()

	if t.Date.Format("02.01.2006") == today.Format("02.01.2006") {
		return true
	}

	switch t.Repeat {
	case "daily":
		return true
	case "weekly":
		todayWeekday := strings.ToLower(today.Weekday().String())
		for _, day := range t.Days {
			if strings.ToLower(day) == todayWeekday {
				return true
			}
		}
	case "custom":
		daysDiff := int(today.Sub(t.Date).Hours() / 24)
		return daysDiff >= 0 && daysDiff%t.Interval == 0
	}
	return false
}
