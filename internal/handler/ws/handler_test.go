package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"messenger/internal/domain"
	wshandler "messenger/internal/handler/ws"
	"messenger/internal/service"
	"messenger/pkg/jwt"
)

func testJWTManager() *jwt.Manager {
	return jwt.NewManager(jwt.Config{
		AccessSecret:  "access-secret",
		RefreshSecret: "refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    168 * time.Hour,
	})
}

type mockUserRepo struct{}
type mockChatRepo struct{}
type mockMessageRepo struct {
	createFn func(ctx context.Context, msg *domain.Message) (*domain.Message, error)
}
type mockMemberRepo struct {
	getFn         func(ctx context.Context, chatID, userID int64) (*domain.ChatMember, error)
	listUserIDsFn func(ctx context.Context, chatID int64) ([]int64, error)
}

func (m *mockUserRepo) Create(context.Context, string, string) (*domain.User, error) { return nil, nil }
func (m *mockUserRepo) GetByLogin(context.Context, string) (*domain.User, error)     { return nil, nil }
func (m *mockUserRepo) GetByID(context.Context, int64) (*domain.User, error)         { return nil, nil }
func (m *mockUserRepo) SearchByLogin(context.Context, string, int64, int) ([]domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateLogin(context.Context, int64, string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdatePasswordHash(context.Context, int64, string) error { return nil }

func (m *mockChatRepo) CreateDirect(context.Context, int64, int64) (*domain.Chat, error) {
	return nil, nil
}
func (m *mockChatRepo) CreateGroup(context.Context, string, int64) (*domain.Chat, error) {
	return nil, nil
}
func (m *mockChatRepo) UpdateChatTitle(context.Context, int64, string) (*domain.Chat, error) {
	return nil, nil
}
func (m *mockChatRepo) GetByID(context.Context, int64) (*domain.Chat, error) { return nil, nil }
func (m *mockChatRepo) GetDirectByUsers(context.Context, int64, int64) (*domain.Chat, error) {
	return nil, nil
}
func (m *mockChatRepo) ListByUser(context.Context, int64) ([]domain.ChatListItem, error) {
	return nil, nil
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	if m.createFn != nil {
		return m.createFn(ctx, msg)
	}
	return nil, nil
}
func (m *mockMessageRepo) ListByChat(context.Context, int64, int64, int) ([]domain.Message, error) {
	return nil, nil
}
func (m *mockMessageRepo) Search(context.Context, int64, string) ([]domain.Message, error) {
	return nil, nil
}

func (m *mockMemberRepo) Add(context.Context, *domain.ChatMember) error { return nil }
func (m *mockMemberRepo) Remove(context.Context, int64, int64) error    { return nil }
func (m *mockMemberRepo) Get(ctx context.Context, chatID, userID int64) (*domain.ChatMember, error) {
	if m.getFn != nil {
		return m.getFn(ctx, chatID, userID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockMemberRepo) ListUserIDs(ctx context.Context, chatID int64) ([]int64, error) {
	if m.listUserIDsFn != nil {
		return m.listUserIDsFn(ctx, chatID)
	}
	return nil, nil
}
func (m *mockMemberRepo) ListByChat(context.Context, int64) ([]domain.ChatMember, error) {
	return nil, nil
}

type mockReadStateRepo struct{}

func (m *mockReadStateRepo) UpsertReadState(context.Context, int64, int64, int64) (int64, error) {
	return 0, nil
}
func (m *mockReadStateRepo) GetReadState(context.Context, int64) ([]domain.ChatReadState, error) {
	return nil, nil
}
func (m *mockReadStateRepo) IsReadByAll(context.Context, int64, int64, int64) (bool, error) {
	return false, nil
}

func newWSTestServer(t *testing.T, messages *mockMessageRepo, members *mockMemberRepo, authTimeout time.Duration) *httptest.Server {
	t.Helper()

	svc := service.New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, nil, testJWTManager())
	hub := wshandler.NewHub()
	handler := wshandler.NewHandler(svc, testJWTManager(), hub, wshandler.Config{AuthTimeout: authTimeout}, nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func TestWSAuthTimeoutClosesConnection(t *testing.T) {
	t.Parallel()

	server := newWSTestServer(t, &mockMessageRepo{}, &mockMemberRepo{}, 200*time.Millisecond)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer resp.Body.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected connection close on auth timeout")
	}
}

func TestWSSendMessageAck(t *testing.T) {
	t.Parallel()

	clientMsgID := uuid.NewString()
	now := time.Now()

	messages := &mockMessageRepo{
		createFn: func(_ context.Context, msg *domain.Message) (*domain.Message, error) {
			return &domain.Message{
				ID:          42,
				ChatID:      msg.ChatID,
				SenderID:    msg.SenderID,
				ClientMsgID: msg.ClientMsgID,
				Body:        msg.Body,
				CreatedAt:   now,
			}, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleMember}, nil
		},
		listUserIDsFn: func(_ context.Context, _ int64) ([]int64, error) {
			return []int64{1, 2}, nil
		},
	}

	server := newWSTestServer(t, messages, members, time.Second)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer resp.Body.Close()
	defer conn.Close()

	token, err := testJWTManager().IssueAccess(1)
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	if err := conn.WriteJSON(map[string]string{"token": token}); err != nil {
		t.Fatalf("auth write: %v", err)
	}

	sendFrame, _ := json.Marshal(map[string]any{
		"type":          wshandler.FrameTypeSendMessage,
		"chat_id":       10,
		"client_msg_id": clientMsgID,
		"body":          "hello",
	})
	if err := conn.WriteMessage(websocket.TextMessage, sendFrame); err != nil {
		t.Fatalf("send write: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}

	var ack struct {
		Type        string `json:"type"`
		ClientMsgID string `json:"client_msg_id"`
		ServerID    int64  `json:"server_id"`
	}
	if err := json.Unmarshal(payload, &ack); err != nil {
		t.Fatalf("unmarshal ack: %v", err)
	}
	if ack.Type != wshandler.FrameTypeAck {
		t.Fatalf("type = %q, want ack", ack.Type)
	}
	if ack.ClientMsgID != clientMsgID {
		t.Fatalf("client_msg_id = %q", ack.ClientMsgID)
	}
	if ack.ServerID != 42 {
		t.Fatalf("server_id = %d, want 42", ack.ServerID)
	}
}
