package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"messenger/internal/service"
	"messenger/pkg/jwt"
)

const defaultAuthTimeout = 5 * time.Second

type Config struct {
	AuthTimeout     time.Duration
	AllowedOrigins  []string
}

func (c Config) authTimeout() time.Duration {
	if c.AuthTimeout > 0 {
		return c.AuthTimeout
	}
	return defaultAuthTimeout
}

type Handler struct {
	svc    *service.Service
	jwt    *jwt.Manager
	hub    *Hub
	cfg    Config
	logger *slog.Logger
}

func NewHandler(svc *service.Service, jwtManager *jwt.Manager, hub *Hub, cfg Config, logger *slog.Logger) *Handler {
	configureUpgrader(cfg.AllowedOrigins)

	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		svc:    svc,
		jwt:    jwtManager,
		hub:    hub,
		cfg:    cfg,
		logger: logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("ws upgrade failed", "err", err)
		return
	}

	conn.SetReadLimit(maxMessageSize)

	authCtx, cancelAuth := context.WithCancel(r.Context())
	defer cancelAuth()

	authDone := make(chan int64, 1)
	authErr := make(chan error, 1)

	go h.authenticate(authCtx, conn, authDone, authErr)

	var userID int64
	select {
	case userID = <-authDone:
	case err = <-authErr:
		h.logger.Debug("ws auth failed", "err", err)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication failed"))
		_ = conn.Close()
		return
	case <-time.After(h.cfg.authTimeout()):
		h.logger.Debug("ws auth timeout")
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication timeout"))
		_ = conn.Close()
		return
	case <-h.hub.Done():
		_ = conn.Close()
		return
	case <-r.Context().Done():
		_ = conn.Close()
		return
	}

	cancelAuth()

	client := newClient(h.hub, conn, userID)
	h.hub.Register(client)
	defer func() {
		h.hub.Unregister(client)
		client.close()
	}()

	go client.writePump()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		select {
		case <-h.hub.Done():
			return
		case <-client.closed:
			return
		default:
		}

		_, payload, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				h.logger.Debug("ws read ended", "err", err)
			}
			return
		}

		if err := h.handleFrame(r.Context(), client, userID, payload); err != nil {
			h.logger.Debug("ws frame handling failed", "err", err)
			return
		}
	}
}

func (h *Handler) authenticate(ctx context.Context, conn *websocket.Conn, authDone chan<- int64, authErr chan<- error) {
	_ = conn.SetReadDeadline(time.Now().Add(h.cfg.authTimeout()))

	_, payload, err := conn.ReadMessage()
	if err != nil {
		select {
		case authErr <- err:
		case <-ctx.Done():
		}
		return
	}

	var frame authFrame
	if err := json.Unmarshal(payload, &frame); err != nil || frame.Token == "" {
		select {
		case authErr <- errors.New("invalid auth frame"):
		case <-ctx.Done():
		}
		return
	}

	userID, err := h.jwt.ParseAccess(frame.Token)
	if err != nil {
		select {
		case authErr <- err:
		case <-ctx.Done():
		}
		return
	}

	select {
	case authDone <- userID:
	case <-ctx.Done():
	}
}

func (h *Handler) handleFrame(ctx context.Context, client *Client, userID int64, payload []byte) error {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return err
	}

	switch envelope.Type {
	case FrameTypeSendMessage:
		return h.handleSendMessage(ctx, client, userID, payload)
	default:
		return errors.New("unknown frame type")
	}
}

func (h *Handler) handleSendMessage(ctx context.Context, client *Client, userID int64, payload []byte) error {
	var frame sendMessageFrame
	if err := json.Unmarshal(payload, &frame); err != nil {
		return err
	}

	msg, err := h.svc.SendMessage(ctx, userID, frame.ChatID, frame.ClientMsgID, frame.Body)
	if err != nil {
		return err
	}

	ack := ackFrame{
		Type:        FrameTypeAck,
		ClientMsgID: msg.ClientMsgID,
		ServerID:    msg.ID,
	}
	if err := client.writeJSONSync(ack); err != nil {
		return err
	}

	memberIDs, err := h.svc.ListChatMemberUserIDs(ctx, frame.ChatID, userID)
	if err != nil {
		h.logger.Warn("ws broadcast skipped", "err", err, "chat_id", frame.ChatID)
		return nil
	}

	push := newMessageFrame{
		Type:   FrameTypeNewMessage,
		ChatID: frame.ChatID,
		Message: messagePayload{
			ID:        msg.ID,
			SenderID:  msg.SenderID,
			Body:      msg.Body,
			CreatedAt: formatTime(msg.CreatedAt),
		},
	}
	pushData, err := json.Marshal(push)
	if err != nil {
		return err
	}

	h.hub.BroadcastNewMessage(frame.ChatID, userID, pushData, memberIDs)
	return nil
}
