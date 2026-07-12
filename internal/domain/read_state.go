package domain

import "time"

// ChatReadState is a per-user read cursor inside a chat.
// Source of truth for receipts; "read by all" is derived from these cursors.
type ChatReadState struct {
	ChatID            int64
	UserID            int64
	LastReadMessageID int64
	UpdatedAt         time.Time
}
