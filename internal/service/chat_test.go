package service

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
)

func TestService_CreateDirectChatReusesExisting(t *testing.T) {
	t.Parallel()

	const (
		callerID int64 = 1
		otherID  int64 = 2
	)

	existing := &domain.Chat{ID: 99, Type: domain.ChatTypeDirect}

	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: id}, nil
		},
	}
	chats := &mockChatRepo{
		createDirectFn: func(_ context.Context, a, b int64) (*domain.Chat, error) {
			if a != callerID || b != otherID {
				t.Fatalf("users = (%d, %d), want (%d, %d)", a, b, callerID, otherID)
			}
			return nil, domain.ErrConflict
		},
		getDirectByUsersFn: func(_ context.Context, a, b int64) (*domain.Chat, error) {
			if a != callerID || b != otherID {
				t.Fatalf("users = (%d, %d), want (%d, %d)", a, b, callerID, otherID)
			}
			return existing, nil
		},
	}

	svc := New(users, chats, &mockMessageRepo{}, &mockMemberRepo{}, testJWTManager())

	chat, err := svc.CreateDirectChat(context.Background(), callerID, otherID)
	if err != nil {
		t.Fatalf("CreateDirectChat: %v", err)
	}
	if chat.ID != existing.ID {
		t.Fatalf("chat id = %d, want %d", chat.ID, existing.ID)
	}
}

func TestService_CreateDirectChatSelfValidation(t *testing.T) {
	t.Parallel()

	svc := New(&mockUserRepo{}, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, testJWTManager())

	_, err := svc.CreateDirectChat(context.Background(), 1, 1)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}
