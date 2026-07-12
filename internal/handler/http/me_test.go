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

func TestGetMeUnauthorized(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	resp, data := env.do(http.MethodGet, "/me", nil, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "unauthorized")
}

func TestUpdateMeUnauthorized(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	resp, data := env.do(http.MethodPatch, "/me", map[string]string{"login": "new"}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "unauthorized")
}

func TestUpdatePasswordUnauthorized(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	resp, data := env.do(http.MethodPatch, "/me/password", map[string]string{
		"current_password": "old",
		"new_password":     "new",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "unauthorized")
}

func TestUpdateMeConflict(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.updateLoginFn = func(_ context.Context, userID int64, login string) (*domain.User, error) {
		if userID != 1 || login != "taken" {
			t.Fatalf("unexpected args: userID=%d login=%q", userID, login)
		}
		return nil, domain.ErrConflict
	}

	resp, data := env.do(http.MethodPatch, "/me", map[string]string{"login": "taken"}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusConflict)
	assertErrorCode(t, data, "conflict")
}

func TestUpdateMeHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	createdAt := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	env.users.updateLoginFn = func(_ context.Context, _ int64, login string) (*domain.User, error) {
		return &domain.User{ID: 1, Login: login, CreatedAt: createdAt}, nil
	}

	resp, data := env.do(http.MethodPatch, "/me", map[string]string{"login": "alice2"}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.ID != 1 || body.Login != "alice2" || body.CreatedAt == "" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestUpdatePasswordWrongCurrent(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	hash, err := password.Hash("correct")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id, Login: "alice", PasswordHash: hash}, nil
	}

	resp, data := env.do(http.MethodPatch, "/me/password", map[string]string{
		"current_password": "wrong",
		"new_password":     "newsecret",
	}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "invalid_credentials")
}

func TestGetMeHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	createdAt := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id, Login: "alice", PasswordHash: "should-not-leak", CreatedAt: createdAt}, nil
	}

	resp, data := env.do(http.MethodGet, "/me", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["login"] != "alice" {
		t.Fatalf("login = %v", body["login"])
	}
	if _, ok := body["password_hash"]; ok {
		t.Fatal("password_hash must not be present")
	}
}
