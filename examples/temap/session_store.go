package main

import (
	"fmt"
	"time"

	"github.com/majiddarvishan/snipgo/temap"
)

type Session struct {
	UserID    string
	Data      map[string]interface{}
	CreatedAt time.Time
}

type SessionStore struct {
	sessions *temap.TimedMap
	ttl      time.Duration
}

func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{
		ttl: ttl,
		sessions: temap.NewWithCapacity(1000, func(key, value any) {
			session := value.(*Session)
			fmt.Printf("Session expired: user=%s, duration=%v\n",
				session.UserID, time.Since(session.CreatedAt))
		}),
	}
}

func (s *SessionStore) Create(sessionID, userID string, data map[string]interface{}) {
	session := &Session{
		UserID:    userID,
		Data:      data,
		CreatedAt: time.Now(),
	}
	s.sessions.SetTemporary(sessionID, session, s.ttl)
	fmt.Printf("Created session %s for user %s\n", sessionID, userID)
}

func (s *SessionStore) Get(sessionID string) (*Session, bool) {
	val, ok := s.sessions.Get(sessionID)
	if !ok {
		return nil, false
	}
	return val.(*Session), true
}

func (s *SessionStore) Extend(sessionID string) bool {
	newExpiry := time.Now().Add(s.ttl)
	return s.sessions.SetExpiry(sessionID, newExpiry)
}

func (s *SessionStore) Delete(sessionID string) bool {
	return s.sessions.Remove(sessionID)
}

func (s *SessionStore) ActiveSessions() int {
	return s.sessions.Size()
}

func main() {
	store := NewSessionStore(5 * time.Second)

	// Create sessions
	store.Create("sess1", "user123", map[string]interface{}{"role": "admin"})
	store.Create("sess2", "user456", map[string]interface{}{"role": "user"})

	fmt.Printf("Active sessions: %d\n", store.ActiveSessions())

	// Get session
	if session, ok := store.Get("sess1"); ok {
		fmt.Printf("Found session for user: %s\n", session.UserID)
	}

	// Extend session
	time.Sleep(3 * time.Second)
	if store.Extend("sess1") {
		fmt.Println("Session extended")
	}

	// Wait for expiration
	time.Sleep(7 * time.Second)
	fmt.Printf("Active sessions after expiration: %d\n", store.ActiveSessions())
}
