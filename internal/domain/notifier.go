package domain

import "context"

// RealtimeNotifier pushes realtime events to online clients.
// Implementations live in the transport layer (WS hub); service must not import handlers.
type RealtimeNotifier interface {
	NotifyRead(ctx context.Context, chatID, userID, lastReadMessageID int64)
}
