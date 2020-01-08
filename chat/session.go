package chat

import "time"

type Session struct {
	Expire time.Time
	UserID string
}
