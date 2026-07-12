package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"messenger/internal/domain"
	"messenger/pkg/jwt"
	"messenger/pkg/password"
)

func testJWTManager() *jwt.Manager {
	return jwt.NewManager(jwt.Config{
		AccessSecret:  "access-secret",
		RefreshSecret: "refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    168 * time.Hour,
	})
}

func TestService_LoginWrongPassword(t *testing.T) {
	t.Parallel()

	hash, err := password.Hash("correct-password")
	if err != nil {
		t.Fatalf("password.Hash: %v", err)
	}

	users := &mockUserRepo{
		getByLoginFn: func(_ context.Context, login string) (*domain.User, error) {
			if login != "alice" {
				t.Fatalf("login = %q, want alice", login)
			}
			return &domain.User{ID: 1, Login: "alice", PasswordHash: hash}, nil
		},
	}

	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	_, _, err = svc.Login(context.Background(), "alice", "wrong-password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_LoginUnknownUser(t *testing.T) {
	t.Parallel()

	users := &mockUserRepo{
		getByLoginFn: func(context.Context, string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	_, _, err := svc.Login(context.Background(), "nobody", "password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_LoginSuccess(t *testing.T) {
	t.Parallel()

	hash, err := password.Hash("secret")
	if err != nil {
		t.Fatalf("password.Hash: %v", err)
	}

	users := &mockUserRepo{
		getByLoginFn: func(context.Context, string) (*domain.User, error) {
			return &domain.User{ID: 42, Login: "alice", PasswordHash: hash}, nil
		},
	}

	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	access, refresh, err := svc.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}

	userID, err := testJWTManager().ParseAccess(access)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if userID != 42 {
		t.Fatalf("userID = %d, want 42", userID)
	}
}

func TestService_RegisterHashesPassword(t *testing.T) {
	t.Parallel()

	var storedHash string
	users := &mockUserRepo{
		createFn: func(_ context.Context, login, passwordHash string) (*domain.User, error) {
			storedHash = passwordHash
			return &domain.User{ID: 1, Login: login}, nil
		},
	}

	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	user, err := svc.Register(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.PasswordHash != "" {
		t.Fatal("Register must not return password hash")
	}
	if storedHash == "" || storedHash == "secret" {
		t.Fatalf("expected argon2 hash, got %q", storedHash)
	}

	ok, err := password.Verify("secret", storedHash)
	if err != nil {
		t.Fatalf("password.Verify: %v", err)
	}
	if !ok {
		t.Fatal("stored hash does not match password")
	}
}
