package storage

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// NotificationStore persists the date (YYYY-MM-DD) of the last successful daily
// price notification. It survives restarts so a catch-up send after the host
// was off/asleep at the scheduled time does not fire twice on the same day.
type NotificationStore struct {
	mu   sync.Mutex
	date string
	path string
}

type notificationState struct {
	LastSent string `json:"last_sent"`
}

func NewNotificationStore(path string) *NotificationStore {
	s := &NotificationStore{path: path}
	s.load()
	return s
}

// LastSent returns the YYYY-MM-DD date of the last successful send, or "".
func (s *NotificationStore) LastSent() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.date
}

// SetLastSent records date as the last successful send and persists it.
func (s *NotificationStore) SetLastSent(date string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.date = date
	s.save()
}

func (s *NotificationStore) save() {
	data, err := json.Marshal(notificationState{LastSent: s.date})
	if err != nil {
		log.Printf("Error marshaling notification state: %v", err)
		return
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		log.Printf("Error writing %s: %v", s.path, err)
	}
}

func (s *NotificationStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading %s: %v", s.path, err)
		}
		return
	}

	var state notificationState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("Error parsing %s: %v", s.path, err)
		return
	}
	s.date = state.LastSent
}
