package service

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
)

func TestService_AddMemberNonAdminForbidden(t *testing.T) {
	t.Parallel()

	const (
		chatID   int64 = 10
		adminID  int64 = 1
		memberID int64 = 2
		newUser  int64 = 3
	)

	chats := &mockChatRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Chat, error) {
			if id != chatID {
				t.Fatalf("chat id = %d, want %d", id, chatID)
			}
			return &domain.Chat{ID: chatID, Type: domain.ChatTypeGroup}, nil
		},
	}

	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			if cID != chatID {
				t.Fatalf("chat id = %d, want %d", cID, chatID)
			}
			switch userID {
			case adminID:
				return &domain.ChatMember{ChatID: chatID, UserID: adminID, Role: domain.RoleMember}, nil
			case memberID:
				return &domain.ChatMember{ChatID: chatID, UserID: memberID, Role: domain.RoleAdmin}, nil
			default:
				return nil, domain.ErrNotFound
			}
		},
		addFn: func(context.Context, *domain.ChatMember) error {
			t.Fatal("Add must not be called for non-admin caller")
			return nil
		},
	}

	svc := New(&mockUserRepo{}, chats, &mockMessageRepo{}, members, &mockReadStateRepo{}, nil, testJWTManager())

	err := svc.AddMember(context.Background(), adminID, chatID, newUser)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("AddMember error = %v, want ErrForbidden", err)
	}
}

func TestService_AddMemberAdminSuccess(t *testing.T) {
	t.Parallel()

	const (
		chatID   int64 = 10
		adminID  int64 = 1
		newUser  int64 = 3
	)

	var added bool
	chats := &mockChatRepo{
		getByIDFn: func(context.Context, int64) (*domain.Chat, error) {
			return &domain.Chat{ID: chatID, Type: domain.ChatTypeGroup}, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, _, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
		},
		addFn: func(_ context.Context, member *domain.ChatMember) error {
			if member.UserID != newUser || member.Role != domain.RoleMember {
				t.Fatalf("unexpected member: %+v", member)
			}
			added = true
			return nil
		},
	}
	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			if id != newUser {
				t.Fatalf("user id = %d, want %d", id, newUser)
			}
			return &domain.User{ID: newUser}, nil
		},
	}

	svc := New(users, chats, &mockMessageRepo{}, members, &mockReadStateRepo{}, nil, testJWTManager())

	if err := svc.AddMember(context.Background(), adminID, chatID, newUser); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if !added {
		t.Fatal("expected member to be added")
	}
}
