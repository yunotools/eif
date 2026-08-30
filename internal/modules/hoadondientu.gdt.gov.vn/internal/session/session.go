package session

import "time"

type Session struct {
	ID        string
	Username  string
	Token     string
	ExpiredAt time.Time
}
