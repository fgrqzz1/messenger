package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"messenger/internal/domain"
	httphandler "messenger/internal/handler/http"
	"messenger/internal/service"
	"messenger/pkg/jwt"
)

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func testJWTManager() *jwt.Manager {
	return jwt.NewManager(jwt.Config{
		AccessSecret:  "access-secret",
		RefreshSecret: "refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    168 * time.Hour,
	})
}

type testEnv struct {
	users      *mockUserRepo
	chats      *mockChatRepo
	messages   *mockMessageRepo
	members    *mockMemberRepo
	readStates *mockReadStateRepo
	svc        *service.Service
	server     *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	env := &testEnv{
		users:      &mockUserRepo{},
		chats:      &mockChatRepo{},
		messages:   &mockMessageRepo{},
		members:    &mockMemberRepo{},
		readStates: &mockReadStateRepo{},
	}
	env.svc = service.New(env.users, env.chats, env.messages, env.members, env.readStates, nil, testJWTManager())
	env.server = httptest.NewServer(httphandler.NewMux(env.svc, testJWTManager()))
	t.Cleanup(env.server.Close)

	return env
}

func (e *testEnv) do(method, path string, body any, token string) (*http.Response, []byte) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		panic(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}
	return resp, data
}

func (e *testEnv) accessToken(t *testing.T, userID int64) string {
	t.Helper()
	token, err := testJWTManager().IssueAccess(userID)
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	return token
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status = %d, want %d", resp.StatusCode, want)
	}
}

func assertErrorCode(t *testing.T, data []byte, want string) {
	t.Helper()
	var er errorResponse
	if err := json.Unmarshal(data, &er); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if er.Error.Code != want {
		t.Fatalf("error code = %q, want %q", er.Error.Code, want)
	}
}

type mockUserRepo struct {
	createFn         func(ctx context.Context, login, passwordHash string) (*domain.User, error)
	getByLoginFn     func(ctx context.Context, login string) (*domain.User, error)
	getByIDFn        func(ctx context.Context, id int64) (*domain.User, error)
	searchByLoginFn  func(ctx context.Context, query string, excludeUserID int64, limit int) ([]domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, login, passwordHash string) (*domain.User, error) {
	return m.createFn(ctx, login, passwordHash)
}
func (m *mockUserRepo) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	return m.getByLoginFn(ctx, login)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockUserRepo) SearchByLogin(ctx context.Context, query string, excludeUserID int64, limit int) ([]domain.User, error) {
	if m.searchByLoginFn != nil {
		return m.searchByLoginFn(ctx, query, excludeUserID, limit)
	}
	return nil, nil
}

type mockChatRepo struct {
	createDirectFn     func(ctx context.Context, userAID, userBID int64) (*domain.Chat, error)
	createGroupFn      func(ctx context.Context, title string, createdBy int64) (*domain.Chat, error)
	getByIDFn          func(ctx context.Context, id int64) (*domain.Chat, error)
	getDirectByUsersFn func(ctx context.Context, userAID, userBID int64) (*domain.Chat, error)
	listByUserFn       func(ctx context.Context, userID int64) ([]domain.ChatListItem, error)
}

func (m *mockChatRepo) CreateDirect(ctx context.Context, userAID, userBID int64) (*domain.Chat, error) {
	return m.createDirectFn(ctx, userAID, userBID)
}
func (m *mockChatRepo) CreateGroup(ctx context.Context, title string, createdBy int64) (*domain.Chat, error) {
	return m.createGroupFn(ctx, title, createdBy)
}
func (m *mockChatRepo) GetByID(ctx context.Context, id int64) (*domain.Chat, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockChatRepo) GetDirectByUsers(ctx context.Context, userAID, userBID int64) (*domain.Chat, error) {
	return m.getDirectByUsersFn(ctx, userAID, userBID)
}
func (m *mockChatRepo) ListByUser(ctx context.Context, userID int64) ([]domain.ChatListItem, error) {
	return m.listByUserFn(ctx, userID)
}

type mockMessageRepo struct {
	createFn     func(ctx context.Context, msg *domain.Message) (*domain.Message, error)
	listByChatFn func(ctx context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error)
	searchFn     func(ctx context.Context, chatID int64, query string) ([]domain.Message, error)
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	return m.createFn(ctx, msg)
}
func (m *mockMessageRepo) ListByChat(ctx context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error) {
	return m.listByChatFn(ctx, chatID, beforeID, limit)
}
func (m *mockMessageRepo) Search(ctx context.Context, chatID int64, query string) ([]domain.Message, error) {
	return m.searchFn(ctx, chatID, query)
}

type mockMemberRepo struct {
	addFn         func(ctx context.Context, member *domain.ChatMember) error
	removeFn      func(ctx context.Context, chatID, userID int64) error
	getFn         func(ctx context.Context, chatID, userID int64) (*domain.ChatMember, error)
	listUserIDsFn func(ctx context.Context, chatID int64) ([]int64, error)
	listByChatFn  func(ctx context.Context, chatID int64) ([]domain.ChatMember, error)
}

func (m *mockMemberRepo) Add(ctx context.Context, member *domain.ChatMember) error {
	return m.addFn(ctx, member)
}
func (m *mockMemberRepo) Remove(ctx context.Context, chatID, userID int64) error {
	return m.removeFn(ctx, chatID, userID)
}
func (m *mockMemberRepo) Get(ctx context.Context, chatID, userID int64) (*domain.ChatMember, error) {
	return m.getFn(ctx, chatID, userID)
}
func (m *mockMemberRepo) ListUserIDs(ctx context.Context, chatID int64) ([]int64, error) {
	if m.listUserIDsFn != nil {
		return m.listUserIDsFn(ctx, chatID)
	}
	return nil, nil
}
func (m *mockMemberRepo) ListByChat(ctx context.Context, chatID int64) ([]domain.ChatMember, error) {
	if m.listByChatFn != nil {
		return m.listByChatFn(ctx, chatID)
	}
	return nil, nil
}

type mockReadStateRepo struct {
	upsertFn func(ctx context.Context, chatID, userID, messageID int64) (int64, error)
	getFn    func(ctx context.Context, chatID int64) ([]domain.ChatReadState, error)
}

func (m *mockReadStateRepo) UpsertReadState(ctx context.Context, chatID, userID, messageID int64) (int64, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, chatID, userID, messageID)
	}
	return messageID, nil
}
func (m *mockReadStateRepo) GetReadState(ctx context.Context, chatID int64) ([]domain.ChatReadState, error) {
	if m.getFn != nil {
		return m.getFn(ctx, chatID)
	}
	return nil, nil
}
func (m *mockReadStateRepo) IsReadByAll(context.Context, int64, int64, int64) (bool, error) {
	return false, nil
}
