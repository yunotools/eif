package session

import "time"

type Session struct {
	ID        string
	Token     string
	ExpiredAt time.Time
}
