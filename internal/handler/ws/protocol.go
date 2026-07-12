package ws

import "time"

const (
	FrameTypeAck         = "ack"
	FrameTypeNewMessage  = "new_message"
	FrameTypeSendMessage = "send_message"
	FrameTypeRead        = "read"
)

type authFrame struct {
	Token string `json:"token"`
}

type sendMessageFrame struct {
	Type        string `json:"type"`
	ChatID      int64  `json:"chat_id"`
	ClientMsgID string `json:"client_msg_id"`
	Body        string `json:"body"`
}

type ackFrame struct {
	Type        string `json:"type"`
	ClientMsgID string `json:"client_msg_id"`
	ServerID    int64  `json:"server_id"`
}

type newMessageFrame struct {
	Type    string         `json:"type"`
	ChatID  int64          `json:"chat_id"`
	Message messagePayload `json:"message"`
}

type readFrame struct {
	Type              string `json:"type"`
	ChatID            int64  `json:"chat_id"`
	UserID            int64  `json:"user_id"`
	LastReadMessageID int64  `json:"last_read_message_id"`
}

type messagePayload struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"sender_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func formatTime(t time.Time) string {
	return t.UTC().Format(timeRFC3339Nano)
}
