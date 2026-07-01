package storage

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// NewsStore persists the URL of the most recent news item already broadcast to
// subscribers. It survives restarts so the background poller never re-sends an
// entry it has already delivered.
type NewsStore struct {
	mu   sync.Mutex
	url  string
	path string
}

type newsState struct {
	LastURL string `json:"last_url"`
}

func NewNewsStore(path string) *NewsStore {
	s := &NewsStore{path: path}
	s.load()
	return s
}

// LastURL returns the URL of the last broadcast item, or "" if none yet.
func (s *NewsStore) LastURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

// SetLastURL records url as the last broadcast item and persists it.
func (s *NewsStore) SetLastURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.url = url
	s.save()
}

func (s *NewsStore) save() {
	data, err := json.Marshal(newsState{LastURL: s.url})
	if err != nil {
		log.Printf("Error marshaling news state: %v", err)
		return
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		log.Printf("Error writing %s: %v", s.path, err)
	}
}

func (s *NewsStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading %s: %v", s.path, err)
		}
		return
	}

	var state newsState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("Error parsing %s: %v", s.path, err)
		return
	}
	s.url = state.LastURL
}
