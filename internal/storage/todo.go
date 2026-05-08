package storage

import (
	"encoding/json"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/VxVxN/telegrambot/internal/model"
)

type TodoStore struct {
	mu     sync.Mutex
	data   map[int]*model.UserTodos
	nextID int
	path   string
}

func NewTodoStore(path string) *TodoStore {
	s := &TodoStore{
		data: make(map[int]*model.UserTodos),
		path: path,
	}
	s.load()
	return s
}

func (s *TodoStore) TodayTodos(userID int) []model.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	ut, ok := s.data[userID]
	if !ok {
		return nil
	}

	var result []model.Todo
	for _, t := range ut.Todos {
		if t.IsDueToday() {
			result = append(result, t)
		}
	}
	return result
}

func (s *TodoStore) AllTodos(userID int) []model.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	ut, ok := s.data[userID]
	if !ok {
		return nil
	}

	result := make([]model.Todo, len(ut.Todos))
	copy(result, ut.Todos)
	return result
}

func (s *TodoStore) Add(userID int, text string, date time.Time) model.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	todo := model.Todo{
		ID:     s.nextID,
		Text:   text,
		Date:   date,
		Repeat: "none",
	}

	if _, ok := s.data[userID]; !ok {
		s.data[userID] = &model.UserTodos{UserID: userID}
	}
	s.data[userID].Todos = append(s.data[userID].Todos, todo)
	s.save()
	return todo
}

func (s *TodoStore) Delete(userID, id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	ut, ok := s.data[userID]
	if !ok {
		return false
	}

	for i, t := range ut.Todos {
		if t.ID == id {
			ut.Todos = slices.Delete(ut.Todos, i, i+1)
			s.save()
			return true
		}
	}
	return false
}

func (s *TodoStore) Clear(userID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ut, ok := s.data[userID]; ok {
		ut.Todos = ut.Todos[:0]
		s.save()
	}
}

func (s *TodoStore) SetRepeat(userID, id int, repeatType string, interval int, days []string) (model.Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ut, ok := s.data[userID]
	if !ok {
		return model.Todo{}, false
	}

	for i := range ut.Todos {
		if ut.Todos[i].ID == id {
			ut.Todos[i].Repeat = repeatType
			ut.Todos[i].Interval = interval
			ut.Todos[i].Days = days
			s.save()
			return ut.Todos[i], true
		}
	}
	return model.Todo{}, false
}

func (s *TodoStore) save() {
	data, err := json.Marshal(s.data)
	if err != nil {
		log.Printf("Error marshaling todos: %v", err)
		return
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		log.Printf("Error writing %s: %v", s.path, err)
	}
}

func (s *TodoStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading %s: %v", s.path, err)
		}
		return
	}

	if err := json.Unmarshal(data, &s.data); err != nil {
		log.Printf("Error parsing %s: %v", s.path, err)
		return
	}

	for _, ut := range s.data {
		for _, t := range ut.Todos {
			if t.ID > s.nextID {
				s.nextID = t.ID
			}
		}
	}
}
