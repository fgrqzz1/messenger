package ws

import (
	"context"
	"encoding/json"
	"log/slog"

	"messenger/internal/domain"
)

// HubReadNotifier adapts Hub to domain.RealtimeNotifier.
type HubReadNotifier struct {
	hub     *Hub
	members domain.MemberRepository
	logger  *slog.Logger
}

func NewHubReadNotifier(hub *Hub, members domain.MemberRepository, logger *slog.Logger) *HubReadNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &HubReadNotifier{hub: hub, members: members, logger: logger}
}

func (n *HubReadNotifier) NotifyRead(ctx context.Context, chatID, userID, lastReadMessageID int64) {
	memberIDs, err := n.members.ListUserIDs(ctx, chatID)
	if err != nil {
		n.logger.Warn("read broadcast skipped", "err", err, "chat_id", chatID)
		return
	}

	frame := readFrame{
		Type:              FrameTypeRead,
		ChatID:            chatID,
		UserID:            userID,
		LastReadMessageID: lastReadMessageID,
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		n.logger.Warn("read broadcast marshal failed", "err", err, "chat_id", chatID)
		return
	}

	n.hub.BroadcastRead(chatID, userID, payload, memberIDs)
}

var _ domain.RealtimeNotifier = (*HubReadNotifier)(nil)
