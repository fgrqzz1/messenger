package postgres

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
)

func TestChatRepository_CreateDirectAndListByUser(t *testing.T) {
	db := newTestDB(t)
	userRepo := NewUserRepository(db)
	chatRepo := NewChatRepository(db)
	msgRepo := NewMessageRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice")
	bobID := createTestUser(t, db, "bob")

	direct, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}
	if direct.Type != domain.ChatTypeDirect {
		t.Fatalf("type = %q, want direct", direct.Type)
	}

	group, err := chatRepo.CreateGroup(ctx, "team", aliceID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if group.Type != domain.ChatTypeGroup {
		t.Fatalf("type = %q, want group", group.Type)
	}

	_, err = msgRepo.Create(ctx, &domain.Message{
		ChatID:      direct.ID,
		SenderID:    aliceID,
		ClientMsgID: "11111111-1111-1111-1111-111111111111",
		Body:        "hello direct",
	})
	if err != nil {
		t.Fatalf("Create direct message: %v", err)
	}

	_, err = msgRepo.Create(ctx, &domain.Message{
		ChatID:      group.ID,
		SenderID:    aliceID,
		ClientMsgID: "22222222-2222-2222-2222-222222222222",
		Body:        "hello group",
	})
	if err != nil {
		t.Fatalf("Create group message: %v", err)
	}

	chats, err := chatRepo.ListByUser(ctx, aliceID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("len(chats) = %d, want 2", len(chats))
	}
	if chats[0].LastMessageBody == nil || *chats[0].LastMessageBody != "hello group" {
		t.Fatalf("expected latest group message first, got %+v", chats[0])
	}

	got, err := chatRepo.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title == nil || *got.Title != "team" {
		t.Fatalf("group title = %+v, want team", got.Title)
	}

	_, err = userRepo.GetByID(ctx, aliceID)
	if err != nil {
		t.Fatalf("user still exists: %v", err)
	}
}

func TestChatRepository_GetDirectByUsers(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice-direct")
	bobID := createTestUser(t, db, "bob-direct")

	created, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}

	got, err := chatRepo.GetDirectByUsers(ctx, bobID, aliceID)
	if err != nil {
		t.Fatalf("GetDirectByUsers reversed: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id = %d, want %d", got.ID, created.ID)
	}

	_, err = chatRepo.CreateDirect(ctx, aliceID, bobID)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate CreateDirect error = %v, want ErrConflict", err)
	}
}
