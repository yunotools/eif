package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("session not found")
	ErrExpired  = errors.New("session expired")
)

type Manager struct {
	mu    sync.RWMutex
	items map[string]*Session
	// Khoảng an toàn trước lúc token thật sự hết hạn
	skew time.Duration
}

func NewManager(skew time.Duration) *Manager {
	return &Manager{
		items: make(map[string]*Session),
		skew:  skew,
	}
}

func (m *Manager) Create(
	token string,
	expiredAt time.Time,
) (*Session, error) {
	id, err := newSessionID()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		ID:        id,
		Token:     token,
		ExpiredAt: expiredAt,
	}

	m.mu.Lock()
	m.deleteExpiredLocked(time.Now())
	m.items[sess.ID] = sess
	m.mu.Unlock()

	sessClone := *sess

	return &sessClone, nil
}

func (m *Manager) Get(id string) (*Session, error) {
	if id == "" {
		return nil, ErrNotFound
	}

	m.mu.RLock()
	sess, ok := m.items[id]
	if ok {
		sessCopy := *sess
		m.mu.RUnlock()
		if time.Now().Add(m.skew).After(sessCopy.ExpiredAt) {
			m.Delete(id)
			return nil, ErrExpired
		}
		return &sessCopy, nil
	}
	m.mu.RUnlock()

	return nil, ErrNotFound
}

func (m *Manager) Delete(id string) {
	m.mu.Lock()
	delete(m.items, id)
	m.mu.Unlock()
}

func (m *Manager) deleteExpiredLocked(now time.Time) {
	threshold := now.Add(m.skew)
	for id, sess := range m.items {
		if threshold.After(sess.ExpiredAt) {
			delete(m.items, id)
		}
	}
}

func newSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
