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

	svc := New(users, chats, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

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

	svc := New(&mockUserRepo{}, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.CreateDirectChat(context.Background(), 1, 1)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestService_UpdateChatTitleNonAdminForbidden(t *testing.T) {
	t.Parallel()

	chats := &mockChatRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Chat, error) {
			return &domain.Chat{ID: id, Type: domain.ChatTypeGroup}, nil
		},
		updateChatTitleFn: func(context.Context, int64, string) (*domain.Chat, error) {
			t.Fatal("UpdateChatTitle must not be called for non-admin")
			return nil, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleMember}, nil
		},
	}

	svc := New(&mockUserRepo{}, chats, &mockMessageRepo{}, members, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.UpdateChatTitle(context.Background(), 2, 1, "new")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestService_UpdateChatTitleDirectValidation(t *testing.T) {
	t.Parallel()

	chats := &mockChatRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Chat, error) {
			return &domain.Chat{ID: id, Type: domain.ChatTypeDirect}, nil
		},
		updateChatTitleFn: func(context.Context, int64, string) (*domain.Chat, error) {
			t.Fatal("UpdateChatTitle must not be called for direct chat")
			return nil, nil
		},
	}

	svc := New(&mockUserRepo{}, chats, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.UpdateChatTitle(context.Background(), 1, 1, "new")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestService_UpdateChatTitleSuccessNotifies(t *testing.T) {
	t.Parallel()

	const (
		adminID int64 = 1
		chatID  int64 = 10
	)
	newTitle := "renamed"

	chats := &mockChatRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Chat, error) {
			return &domain.Chat{ID: id, Type: domain.ChatTypeGroup}, nil
		},
		updateChatTitleFn: func(_ context.Context, id int64, title string) (*domain.Chat, error) {
			if id != chatID || title != newTitle {
				t.Fatalf("UpdateChatTitle(%d, %q)", id, title)
			}
			return &domain.Chat{ID: id, Type: domain.ChatTypeGroup, Title: &newTitle}, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: cID, UserID: userID, Role: domain.RoleAdmin}, nil
		},
	}
	notifier := &mockRealtimeNotifier{}

	svc := New(&mockUserRepo{}, chats, &mockMessageRepo{}, members, &mockReadStateRepo{}, notifier, testJWTManager())

	chat, err := svc.UpdateChatTitle(context.Background(), adminID, chatID, "  "+newTitle+"  ")
	if err != nil {
		t.Fatalf("UpdateChatTitle: %v", err)
	}
	if chat.Title == nil || *chat.Title != newTitle {
		t.Fatalf("title = %+v, want %q", chat.Title, newTitle)
	}
	if len(notifier.chatUpdatedCalls) != 1 {
		t.Fatalf("chatUpdatedCalls = %d, want 1", len(notifier.chatUpdatedCalls))
	}
	call := notifier.chatUpdatedCalls[0]
	if call.chatID != chatID || call.actorUserID != adminID || call.title != newTitle {
		t.Fatalf("unexpected notify call: %+v", call)
	}
}
