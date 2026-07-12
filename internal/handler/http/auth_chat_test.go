package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"messenger/internal/domain"
	"messenger/pkg/password"
)

func TestRegisterHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.createFn = func(_ context.Context, login, _ string) (*domain.User, error) {
		return &domain.User{ID: 1, Login: login, CreatedAt: time.Now()}, nil
	}

	resp, data := env.do(http.MethodPost, "/register", map[string]string{
		"login":    "alice",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["login"] != "alice" {
		t.Fatalf("login = %v", body["login"])
	}
}

func TestRegisterValidationError(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodPost, "/register", map[string]string{
		"login":    "",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func TestLoginHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	hash, err := passwordHash("secret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	env.users.getByLoginFn = func(_ context.Context, login string) (*domain.User, error) {
		return &domain.User{ID: 1, Login: login, PasswordHash: hash}, nil
	}

	resp, data := env.do(http.MethodPost, "/login", map[string]string{
		"login":    "alice",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.AccessToken == "" || body.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.getByLoginFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, domain.ErrNotFound
	}

	resp, data := env.do(http.MethodPost, "/login", map[string]string{
		"login":    "alice",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "invalid_credentials")
}

func TestRefreshHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id, Login: "alice"}, nil
	}

	pair, err := testJWTManager().IssuePair(1)
	if err != nil {
		t.Fatalf("IssuePair: %v", err)
	}

	resp, data := env.do(http.MethodPost, "/refresh", nil, pair.RefreshToken)
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestRefreshUnauthorized(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodPost, "/refresh", nil, "bad-token")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "unauthorized")
}

func TestListChatsHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	now := time.Now()
	title := "team"
	env.chats.listByUserFn = func(_ context.Context, userID int64) ([]domain.ChatListItem, error) {
		if userID != 1 {
			t.Fatalf("userID = %d", userID)
		}
		return []domain.ChatListItem{{
			ID:    10,
			Type:  domain.ChatTypeGroup,
			Title: &title,
			LastMessageAt: func() *time.Time {
				v := now
				return &v
			}(),
		}}, nil
	}

	resp, _ := env.do(http.MethodGet, "/chats", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)
}

func TestListChatsUnauthorized(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodGet, "/chats", nil, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "unauthorized")
}

func TestCreateChatDirectHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id}, nil
	}
	env.chats.createDirectFn = func(_ context.Context, a, b int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 5, Type: domain.ChatTypeDirect, UserAID: &a, UserBID: &b, CreatedAt: time.Now()}, nil
	}

	resp, _ := env.do(http.MethodPost, "/chats", map[string]any{
		"type":    "direct",
		"user_id": 2,
	}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusCreated)
}

func TestCreateChatValidationError(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodPost, "/chats", map[string]string{
		"type": "group",
	}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func TestListMessagesHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleMember}, nil
	}
	env.messages.listByChatFn = func(_ context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error) {
		return []domain.Message{{ID: 100, SenderID: 1, Body: "hi", CreatedAt: time.Now()}}, nil
	}

	resp, _ := env.do(http.MethodGet, "/chats/1/messages?limit=10", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)
}

func TestListMessagesForbidden(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.members.getFn = func(_ context.Context, _, _ int64) (*domain.ChatMember, error) {
		return nil, domain.ErrNotFound
	}

	resp, data := env.do(http.MethodGet, "/chats/1/messages", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusForbidden)
	assertErrorCode(t, data, "forbidden")
}

func TestAddMemberHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		if userID == 1 {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
		}
		return nil, domain.ErrNotFound
	}
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id}, nil
	}
	env.members.addFn = func(_ context.Context, _ *domain.ChatMember) error { return nil }

	resp, _ := env.do(http.MethodPost, "/chats/1/members", map[string]int64{"user_id": 2}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusNoContent)
}

func TestAddMemberForbidden(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, _, _ int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{Role: domain.RoleMember}, nil
	}

	resp, data := env.do(http.MethodPost, "/chats/1/members", map[string]int64{"user_id": 2}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusForbidden)
	assertErrorCode(t, data, "forbidden")
}

func TestRemoveMemberHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
	}
	env.members.removeFn = func(_ context.Context, _, _ int64) error { return nil }

	resp, _ := env.do(http.MethodDelete, "/chats/1/members/2", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusNoContent)
}

func TestRemoveMemberNotFound(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
	}
	env.members.removeFn = func(_ context.Context, _, _ int64) error {
		return domain.ErrNotFound
	}

	resp, data := env.do(http.MethodDelete, "/chats/1/members/2", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, data, "not_found")
}

func TestUpdateChatTitleHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	currentTitle := "old"
	env.chats.getByIDFn = func(_ context.Context, id int64) (*domain.Chat, error) {
		return &domain.Chat{ID: id, Type: domain.ChatTypeGroup, Title: &currentTitle, CreatedAt: time.Now()}, nil
	}
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
	}
	env.chats.updateChatTitleFn = func(_ context.Context, chatID int64, title string) (*domain.Chat, error) {
		currentTitle = title
		return &domain.Chat{ID: chatID, Type: domain.ChatTypeGroup, Title: &currentTitle, CreatedAt: time.Now()}, nil
	}
	env.chats.listByUserFn = func(_ context.Context, _ int64) ([]domain.ChatListItem, error) {
		title := currentTitle
		return []domain.ChatListItem{{
			ID:    1,
			Type:  domain.ChatTypeGroup,
			Title: &title,
		}}, nil
	}

	token := env.accessToken(t, 1)
	resp, data := env.do(http.MethodPatch, "/chats/1", map[string]string{"title": "новое название"}, token)
	assertStatus(t, resp, http.StatusOK)

	var patched struct {
		Title *string `json:"title"`
	}
	if err := json.Unmarshal(data, &patched); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if patched.Title == nil || *patched.Title != "новое название" {
		t.Fatalf("title = %+v, want новое название", patched.Title)
	}

	resp, data = env.do(http.MethodGet, "/chats", nil, token)
	assertStatus(t, resp, http.StatusOK)

	var list []struct {
		ID    int64   `json:"id"`
		Title *string `json:"title"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) != 1 || list[0].Title == nil || *list[0].Title != "новое название" {
		t.Fatalf("GET /chats = %+v, want updated title", list)
	}
}

func TestUpdateChatTitleForbidden(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, _, _ int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{Role: domain.RoleMember}, nil
	}
	env.chats.updateChatTitleFn = func(context.Context, int64, string) (*domain.Chat, error) {
		t.Fatal("UpdateChatTitle must not be called for non-admin")
		return nil, nil
	}

	resp, data := env.do(http.MethodPatch, "/chats/1", map[string]string{"title": "x"}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusForbidden)
	assertErrorCode(t, data, "forbidden")
}

func TestUpdateChatTitleDirectForbidden(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeDirect}, nil
	}
	env.chats.updateChatTitleFn = func(context.Context, int64, string) (*domain.Chat, error) {
		t.Fatal("UpdateChatTitle must not be called for direct chat")
		return nil, nil
	}

	resp, data := env.do(http.MethodPatch, "/chats/1", map[string]string{"title": "x"}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func TestSearchHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID}, nil
	}
	env.messages.searchFn = func(_ context.Context, _ int64, query string) ([]domain.Message, error) {
		if query != "hello" {
			t.Fatalf("query = %q", query)
		}
		return []domain.Message{{ID: 1, SenderID: 1, Body: "hello", CreatedAt: time.Now()}}, nil
	}

	resp, _ := env.do(http.MethodGet, "/chats/1/search?q=hello", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)
}

func TestSearchValidationError(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodGet, "/chats/1/search", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func TestSearchUsersHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.searchByLoginFn = func(_ context.Context, query string, excludeUserID int64, limit int) ([]domain.User, error) {
		if query != "ali" {
			t.Fatalf("query = %q, want ali", query)
		}
		if excludeUserID != 1 {
			t.Fatalf("excludeUserID = %d, want 1", excludeUserID)
		}
		if limit != 20 {
			t.Fatalf("limit = %d, want 20", limit)
		}
		return []domain.User{{ID: 2, Login: "alice"}}, nil
	}

	resp, data := env.do(http.MethodGet, "/users/search?login=ali", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)

	var got []struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].ID != 2 || got[0].Login != "alice" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestSearchUsersExcludesCaller(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.searchByLoginFn = func(_ context.Context, _ string, excludeUserID int64, _ int) ([]domain.User, error) {
		if excludeUserID != 42 {
			t.Fatalf("excludeUserID = %d, want 42", excludeUserID)
		}
		// Repository must not return the caller; mock mirrors that contract.
		return []domain.User{{ID: 7, Login: "bobbie"}}, nil
	}

	resp, data := env.do(http.MethodGet, "/users/search?login=bo", nil, env.accessToken(t, 42))
	assertStatus(t, resp, http.StatusOK)

	var got []struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, u := range got {
		if u.ID == 42 {
			t.Fatalf("caller id %d present in results: %+v", 42, got)
		}
	}
	if len(got) != 1 || got[0].Login != "bobbie" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestSearchUsersShortQuery(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.searchByLoginFn = func(context.Context, string, int64, int) ([]domain.User, error) {
		t.Fatal("SearchByLogin must not be called for short query")
		return nil, nil
	}

	resp, data := env.do(http.MethodGet, "/users/search?login=a", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func passwordHash(raw string) (string, error) {
	return password.Hash(raw)
}
