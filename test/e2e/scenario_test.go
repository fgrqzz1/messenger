package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log/slog"

	httphandler "messenger/internal/handler/http"
	wshandler "messenger/internal/handler/ws"
	"messenger/internal/repository/postgres"
	"messenger/internal/service"
	"messenger/pkg/jwt"
)

const (
	testAccessSecret  = "e2e-access-secret"
	testRefreshSecret = "e2e-refresh-secret"
)

type e2eEnv struct {
	t      *testing.T
	server *httptest.Server
	db     *postgres.DB
	jwt    *jwt.Manager
}

func newE2EEnv(t *testing.T) *e2eEnv {
	t.Helper()

	db := postgres.SetupTestDB(t)

	jwtManager := jwt.NewManager(jwt.Config{
		AccessSecret:  testAccessSecret,
		RefreshSecret: testRefreshSecret,
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    168 * time.Hour,
	})

	userRepo := postgres.NewUserRepository(db)
	chatRepo := postgres.NewChatRepository(db)
	messageRepo := postgres.NewMessageRepository(db)
	memberRepo := postgres.NewMemberRepository(db)

	svc := service.New(userRepo, chatRepo, messageRepo, memberRepo, jwtManager)

	hub := wshandler.NewHub()
	wsHandler := wshandler.NewHandler(svc, jwtManager, hub, wshandler.Config{}, slog.Default())

	mux := http.NewServeMux()
	mux.Handle("GET /ws", wsHandler)
	mux.Handle("/", httphandler.NewMux(svc, jwtManager))

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
		hub.Shutdown(context.Background())
	})

	return &e2eEnv{t: t, server: server, db: db, jwt: jwtManager}
}

func (e *e2eEnv) baseURL() string {
	return e.server.URL
}

func (e *e2eEnv) wsURL() string {
	return "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
}

func (e *e2eEnv) do(method, path string, body any, token string) (*http.Response, []byte) {
	e.t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, e.baseURL()+path, reader)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("do request: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		e.t.Fatalf("read body: %v", err)
	}
	return resp, data
}

func (e *e2eEnv) register(login, password string) int64 {
	e.t.Helper()

	resp, data := e.do(http.MethodPost, "/register", map[string]string{
		"login":    login,
		"password": password,
	}, "")
	if resp.StatusCode != http.StatusCreated {
		e.t.Fatalf("register %q: status = %d, body = %s", login, resp.StatusCode, data)
	}

	var body struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		e.t.Fatalf("unmarshal register: %v", err)
	}
	return body.ID
}

func (e *e2eEnv) login(login, password string) (access, refresh string) {
	e.t.Helper()

	resp, data := e.do(http.MethodPost, "/login", map[string]string{
		"login":    login,
		"password": password,
	}, "")
	if resp.StatusCode != http.StatusOK {
		e.t.Fatalf("login %q: status = %d, body = %s", login, resp.StatusCode, data)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		e.t.Fatalf("unmarshal login: %v", err)
	}
	if body.AccessToken == "" || body.RefreshToken == "" {
		e.t.Fatal("expected access and refresh tokens")
	}
	return body.AccessToken, body.RefreshToken
}

func (e *e2eEnv) createGroupChat(token, title string) int64 {
	e.t.Helper()

	resp, data := e.do(http.MethodPost, "/chats", map[string]string{
		"type":  "group",
		"title": title,
	}, token)
	if resp.StatusCode != http.StatusCreated {
		e.t.Fatalf("create group chat: status = %d, body = %s", resp.StatusCode, data)
	}

	var body struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		e.t.Fatalf("unmarshal chat: %v", err)
	}
	return body.ID
}

func (e *e2eEnv) addMember(token string, chatID, userID int64) {
	e.t.Helper()

	resp, data := e.do(http.MethodPost, fmt.Sprintf("/chats/%d/members", chatID), map[string]int64{
		"user_id": userID,
	}, token)
	if resp.StatusCode != http.StatusNoContent {
		e.t.Fatalf("add member: status = %d, body = %s", resp.StatusCode, data)
	}
}

func (e *e2eEnv) removeMember(token string, chatID, userID int64) int {
	e.t.Helper()

	resp, _ := e.do(http.MethodDelete, fmt.Sprintf("/chats/%d/members/%d", chatID, userID), nil, token)
	return resp.StatusCode
}

func (e *e2eEnv) listMessages(token string, chatID int64, beforeID int64, limit int) []messageDTO {
	e.t.Helper()

	path := fmt.Sprintf("/chats/%d/messages?limit=%d", chatID, limit)
	if beforeID > 0 {
		path = fmt.Sprintf("/chats/%d/messages?before_id=%d&limit=%d", chatID, beforeID, limit)
	}

	resp, data := e.do(http.MethodGet, path, nil, token)
	if resp.StatusCode != http.StatusOK {
		e.t.Fatalf("list messages: status = %d, body = %s", resp.StatusCode, data)
	}

	var messages []messageDTO
	if err := json.Unmarshal(data, &messages); err != nil {
		e.t.Fatalf("unmarshal messages: %v", err)
	}
	return messages
}

func (e *e2eEnv) searchMessages(token string, chatID int64, query string) []messageDTO {
	e.t.Helper()

	path := fmt.Sprintf("/chats/%d/search?q=%s", chatID, query)
	resp, data := e.do(http.MethodGet, path, nil, token)
	if resp.StatusCode != http.StatusOK {
		e.t.Fatalf("search messages: status = %d, body = %s", resp.StatusCode, data)
	}

	var messages []messageDTO
	if err := json.Unmarshal(data, &messages); err != nil {
		e.t.Fatalf("unmarshal search: %v", err)
	}
	return messages
}

func (e *e2eEnv) messageCount(chatID int64) int {
	e.t.Helper()

	var count int
	err := e.db.Pool().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM messages WHERE chat_id = $1", chatID,
	).Scan(&count)
	if err != nil {
		e.t.Fatalf("count messages: %v", err)
	}
	return count
}

func (e *e2eEnv) connectWS(token string) *websocket.Conn {
	e.t.Helper()

	conn, resp, err := websocket.DefaultDialer.Dial(e.wsURL(), nil)
	if err != nil {
		e.t.Fatalf("ws dial: %v", err)
	}
	resp.Body.Close()

	if err := conn.WriteJSON(map[string]string{"token": token}); err != nil {
		e.t.Fatalf("ws auth write: %v", err)
	}
	return conn
}

func wsSendMessage(conn *websocket.Conn, chatID int64, clientMsgID, body string) error {
	frame, err := json.Marshal(map[string]any{
		"type":          wshandler.FrameTypeSendMessage,
		"chat_id":       chatID,
		"client_msg_id": clientMsgID,
		"body":          body,
	})
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, frame)
}

func wsReadAck(t *testing.T, conn *websocket.Conn) wsAck {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}

	var ack wsAck
	if err := json.Unmarshal(payload, &ack); err != nil {
		t.Fatalf("unmarshal ack: %v", err)
	}
	if ack.Type != wshandler.FrameTypeAck {
		t.Fatalf("frame type = %q, want ack", ack.Type)
	}
	return ack
}

func wsReadNewMessage(t *testing.T, conn *websocket.Conn) wsNewMessage {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read new_message: %v", err)
	}

	var push wsNewMessage
	if err := json.Unmarshal(payload, &push); err != nil {
		t.Fatalf("unmarshal new_message: %v", err)
	}
	if push.Type != wshandler.FrameTypeNewMessage {
		t.Fatalf("frame type = %q, want new_message", push.Type)
	}
	return push
}

type messageDTO struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"sender_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type wsAck struct {
	Type        string `json:"type"`
	ClientMsgID string `json:"client_msg_id"`
	ServerID    int64  `json:"server_id"`
}

type wsNewMessage struct {
	Type    string `json:"type"`
	ChatID  int64  `json:"chat_id"`
	Message struct {
		ID       int64  `json:"id"`
		SenderID int64  `json:"sender_id"`
		Body     string `json:"body"`
	} `json:"message"`
}

func TestE2EScenario(t *testing.T) {
	env := newE2EEnv(t)

	suffix := uuid.NewString()[:8]
	aliceLogin := "alice_" + suffix
	bobLogin := "bob_" + suffix
	password := "secret123"

	// 1. Register two users
	aliceID := env.register(aliceLogin, password)
	bobID := env.register(bobLogin, password)
	if aliceID <= 0 || bobID <= 0 {
		t.Fatalf("unexpected user ids: alice=%d bob=%d", aliceID, bobID)
	}

	// 2. Login both, obtain access/refresh tokens
	aliceAccess, aliceRefresh := env.login(aliceLogin, password)
	bobAccess, bobRefresh := env.login(bobLogin, password)
	if aliceRefresh == "" || bobRefresh == "" {
		t.Fatal("expected refresh tokens for both users")
	}

	// 3. Alice creates group chat and adds Bob
	chatID := env.createGroupChat(aliceAccess, "e2e-team")
	env.addMember(aliceAccess, chatID, bobID)

	// 4. Both open WS connections and authenticate with first frame
	aliceWS := env.connectWS(aliceAccess)
	defer aliceWS.Close()
	bobWS := env.connectWS(bobAccess)
	defer bobWS.Close()

	// 5. Alice sends message — Bob receives push, Alice receives ack
	clientMsgID := uuid.NewString()
	msgBody := "Hello e2e_search_marker from Alice"

	pushCh := make(chan wsNewMessage, 1)
	go func() {
		pushCh <- wsReadNewMessage(t, bobWS)
	}()

	if err := wsSendMessage(aliceWS, chatID, clientMsgID, msgBody); err != nil {
		t.Fatalf("send message: %v", err)
	}

	ack := wsReadAck(t, aliceWS)
	if ack.ClientMsgID != clientMsgID {
		t.Fatalf("ack client_msg_id = %q, want %q", ack.ClientMsgID, clientMsgID)
	}
	if ack.ServerID <= 0 {
		t.Fatalf("ack server_id = %d, want > 0", ack.ServerID)
	}

	select {
	case push := <-pushCh:
		if push.ChatID != chatID {
			t.Fatalf("push chat_id = %d, want %d", push.ChatID, chatID)
		}
		if push.Message.SenderID != aliceID {
			t.Fatalf("push sender_id = %d, want %d", push.Message.SenderID, aliceID)
		}
		if push.Message.Body != msgBody {
			t.Fatalf("push body = %q, want %q", push.Message.Body, msgBody)
		}
		if push.Message.ID != ack.ServerID {
			t.Fatalf("push message id = %d, want %d", push.Message.ID, ack.ServerID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for new_message push")
	}

	// 6. Resend same client_msg_id — no duplicate in DB
	countBefore := env.messageCount(chatID)
	if countBefore != 1 {
		t.Fatalf("message count before resend = %d, want 1", countBefore)
	}

	if err := wsSendMessage(aliceWS, chatID, clientMsgID, msgBody); err != nil {
		t.Fatalf("resend message: %v", err)
	}
	ack2 := wsReadAck(t, aliceWS)
	if ack2.ServerID != ack.ServerID {
		t.Fatalf("resend server_id = %d, want %d", ack2.ServerID, ack.ServerID)
	}
	if env.messageCount(chatID) != countBefore {
		t.Fatalf("duplicate message created in DB: count = %d, want %d", env.messageCount(chatID), countBefore)
	}

	// 7. History pagination (before_id) for both participants
	extraBodies := []string{"pagination-msg-2", "pagination-msg-3"}
	for _, body := range extraBodies {
		if err := wsSendMessage(aliceWS, chatID, uuid.NewString(), body); err != nil {
			t.Fatalf("send pagination message: %v", err)
		}
		_ = wsReadAck(t, aliceWS)
	}

	for _, token := range []string{aliceAccess, bobAccess} {
		page1 := env.listMessages(token, chatID, 0, 2)
		if len(page1) != 2 {
			t.Fatalf("page1 len = %d, want 2", len(page1))
		}
		if page1[0].ID <= page1[1].ID {
			t.Fatalf("page1 not ordered by id desc: %d, %d", page1[0].ID, page1[1].ID)
		}

		page2 := env.listMessages(token, chatID, page1[1].ID, 10)
		if len(page2) < 1 {
			t.Fatal("page2 expected at least 1 message")
		}
		for _, m := range page2 {
			if m.ID >= page1[1].ID {
				t.Fatalf("page2 message id %d not before cursor %d", m.ID, page1[1].ID)
			}
		}
	}

	// 8. Search by substring in chat
	found := env.searchMessages(aliceAccess, chatID, "e2e_search_marker")
	if len(found) == 0 {
		t.Fatal("search returned no results")
	}
	var searchHit bool
	for _, m := range found {
		if strings.Contains(m.Body, "e2e_search_marker") {
			searchHit = true
			break
		}
	}
	if !searchHit {
		t.Fatal("search result does not contain expected substring")
	}

	// 9. Non-admin cannot remove member; admin can
	if status := env.removeMember(bobAccess, chatID, aliceID); status != http.StatusForbidden {
		t.Fatalf("non-admin remove status = %d, want 403", status)
	}
	if status := env.removeMember(aliceAccess, chatID, bobID); status != http.StatusNoContent {
		t.Fatalf("admin remove status = %d, want 204", status)
	}
}

func TestE2EInvalidInputNoPanic(t *testing.T) {
	env := newE2EEnv(t)

	req, err := http.NewRequest(http.MethodPost, env.baseURL()+"/register", strings.NewReader("{invalid"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
