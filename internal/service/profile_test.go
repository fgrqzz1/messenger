package service

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
	"messenger/pkg/password"
)

func TestService_UpdateLoginConflict(t *testing.T) {
	t.Parallel()

	users := &mockUserRepo{
		updateLoginFn: func(_ context.Context, userID int64, login string) (*domain.User, error) {
			if userID != 1 || login != "taken" {
				t.Fatalf("unexpected args: %d %q", userID, login)
			}
			return nil, domain.ErrConflict
		},
	}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.UpdateLogin(context.Background(), 1, "taken")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
}

func TestService_UpdatePasswordWrongCurrent(t *testing.T) {
	t.Parallel()

	hash, err := password.Hash("correct")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: id, Login: "alice", PasswordHash: hash}, nil
		},
		updatePasswordHashFn: func(context.Context, int64, string) error {
			t.Fatal("UpdatePasswordHash must not be called")
			return nil
		},
	}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	err = svc.UpdatePassword(context.Background(), 1, "wrong", "newsecret")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_UpdatePasswordSuccess(t *testing.T) {
	t.Parallel()

	hash, err := password.Hash("oldsecret")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	var stored string
	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: id, Login: "alice", PasswordHash: hash}, nil
		},
		updatePasswordHashFn: func(_ context.Context, userID int64, passwordHash string) error {
			if userID != 1 {
				t.Fatalf("userID = %d", userID)
			}
			stored = passwordHash
			return nil
		},
	}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	if err := svc.UpdatePassword(context.Background(), 1, "oldsecret", "newsecret"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}
	ok, err := password.Verify("newsecret", stored)
	if err != nil || !ok {
		t.Fatalf("stored hash does not match new password: ok=%v err=%v", ok, err)
	}
}

func TestService_GetMeStripsPasswordHash(t *testing.T) {
	t.Parallel()

	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: id, Login: "alice", PasswordHash: "secret-hash"}, nil
		},
	}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	user, err := svc.GetMe(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if user.PasswordHash != "" {
		t.Fatal("PasswordHash must be cleared")
	}
}
