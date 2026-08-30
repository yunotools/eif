package dto

import "time"

type SessionResponse struct {
	SessionID        string    `json:"session_id"`
	Username         string    `json:"username"`
	ExpiredAt        time.Time `json:"expired_at"`
	RemainingSeconds int64     `json:"remaining_seconds"`
}
