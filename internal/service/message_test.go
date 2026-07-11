package service

import (
	"context"
	"testing"

	"messenger/internal/domain"
)

func TestService_SendMessageIdempotentByClientMsgID(t *testing.T) {
	t.Parallel()

	const (
		chatID   int64 = 5
		callerID int64 = 1
	)

	clientMsgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	stored := &domain.Message{
		ID:          100,
		ChatID:      chatID,
		SenderID:    callerID,
		ClientMsgID: clientMsgID,
		Body:        "hello",
	}

	createCalls := 0
	messages := &mockMessageRepo{
		createFn: func(_ context.Context, msg *domain.Message) (*domain.Message, error) {
			createCalls++
			if msg.ClientMsgID != clientMsgID {
				t.Fatalf("client_msg_id = %q, want %q", msg.ClientMsgID, clientMsgID)
			}
			return stored, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: cID, UserID: userID, Role: domain.RoleMember}, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, testJWTManager())

	first, err := svc.SendMessage(context.Background(), callerID, chatID, clientMsgID, "hello")
	if err != nil {
		t.Fatalf("first SendMessage: %v", err)
	}

	second, err := svc.SendMessage(context.Background(), callerID, chatID, clientMsgID, "hello")
	if err != nil {
		t.Fatalf("second SendMessage: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("idempotent ids differ: %d vs %d", first.ID, second.ID)
	}
	if createCalls != 2 {
		t.Fatalf("Create calls = %d, want 2 (service delegates idempotency to repository)", createCalls)
	}
}

func TestService_SendMessageNotMemberForbidden(t *testing.T) {
	t.Parallel()

	members := &mockMemberRepo{
		getFn: func(context.Context, int64, int64) (*domain.ChatMember, error) {
			return nil, domain.ErrNotFound
		},
	}
	messages := &mockMessageRepo{
		createFn: func(context.Context, *domain.Message) (*domain.Message, error) {
			t.Fatal("Create must not be called for non-member")
			return nil, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, testJWTManager())

	_, err := svc.SendMessage(context.Background(), 1, 2, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "hi")
	if err != domain.ErrForbidden {
		t.Fatalf("SendMessage error = %v, want ErrForbidden", err)
	}
}
