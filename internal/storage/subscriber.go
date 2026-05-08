package storage

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type SubscriberStore struct {
	mu   sync.Mutex
	data map[int64]bool
	path string
}

func NewSubscriberStore(path string) *SubscriberStore {
	s := &SubscriberStore{
		data: make(map[int64]bool),
		path: path,
	}
	s.load()
	return s
}

// Subscribe returns false if already subscribed.
func (s *SubscriberStore) Subscribe(userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[userID] {
		return false
	}
	s.data[userID] = true
	s.save()
	return true
}

// Unsubscribe returns false if was not subscribed.
func (s *SubscriberStore) Unsubscribe(userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.data[userID] {
		return false
	}
	delete(s.data, userID)
	s.save()
	return true
}

// List returns a snapshot of all subscriber IDs.
func (s *SubscriberStore) List() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]int64, 0, len(s.data))
	for id := range s.data {
		ids = append(ids, id)
	}
	return ids
}

// Remove deletes a subscriber without checking existence (for blocked users).
func (s *SubscriberStore) Remove(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, userID)
	s.save()
}

func (s *SubscriberStore) save() {
	data, err := json.Marshal(s.data)
	if err != nil {
		log.Printf("Error marshaling subscribers: %v", err)
		return
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		log.Printf("Error writing %s: %v", s.path, err)
	}
}

func (s *SubscriberStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading %s: %v", s.path, err)
		}
		return
	}

	if err := json.Unmarshal(data, &s.data); err != nil {
		log.Printf("Error parsing %s: %v", s.path, err)
	}
}
