package service

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
)

func TestService_MarkReadForbiddenForNonMember(t *testing.T) {
	t.Parallel()

	members := &mockMemberRepo{
		getFn: func(_ context.Context, _, _ int64) (*domain.ChatMember, error) {
			return nil, domain.ErrNotFound
		},
	}
	readStates := &mockReadStateRepo{
		upsertFn: func(_ context.Context, _, _, _ int64) (int64, error) {
			t.Fatal("UpsertReadState must not be called for non-member")
			return 0, nil
		},
	}
	notifier := &mockRealtimeNotifier{}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, &mockMessageRepo{}, members, readStates, notifier, testJWTManager())

	err := svc.MarkRead(context.Background(), 1, 99, 10)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("notifier calls = %d, want 0", len(notifier.calls))
	}
}

func TestService_MarkReadBroadcastsEffectiveCursor(t *testing.T) {
	t.Parallel()

	members := &mockMemberRepo{
		getFn: func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleMember}, nil
		},
	}
	readStates := &mockReadStateRepo{
		upsertFn: func(_ context.Context, _, _, messageID int64) (int64, error) {
			return messageID, nil
		},
	}
	notifier := &mockRealtimeNotifier{}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, &mockMessageRepo{}, members, readStates, notifier, testJWTManager())

	if err := svc.MarkRead(context.Background(), 7, 3, 42); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notifier calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.chatID != 7 || call.userID != 3 || call.lastReadMessageID != 42 {
		t.Fatalf("unexpected notify call: %+v", call)
	}
}

func TestService_GetReadStateForbiddenForNonMember(t *testing.T) {
	t.Parallel()

	members := &mockMemberRepo{
		getFn: func(_ context.Context, _, _ int64) (*domain.ChatMember, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, &mockMessageRepo{}, members, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.GetReadState(context.Background(), 1, 99)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}
