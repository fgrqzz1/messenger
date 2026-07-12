package postgres

import (
	"context"
	"testing"

	"messenger/internal/domain"
)

func TestReadStateRepository_UpsertNeverMovesBackward(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	readRepo := NewReadStateRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice-read")
	bobID := createTestUser(t, db, "bob-read")

	chat, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}

	got, err := readRepo.UpsertReadState(ctx, chat.ID, aliceID, 10)
	if err != nil {
		t.Fatalf("UpsertReadState 10: %v", err)
	}
	if got != 10 {
		t.Fatalf("effective = %d, want 10", got)
	}

	got, err = readRepo.UpsertReadState(ctx, chat.ID, aliceID, 5)
	if err != nil {
		t.Fatalf("UpsertReadState 5: %v", err)
	}
	if got != 10 {
		t.Fatalf("effective after lower id = %d, want 10", got)
	}

	got, err = readRepo.UpsertReadState(ctx, chat.ID, aliceID, 15)
	if err != nil {
		t.Fatalf("UpsertReadState 15: %v", err)
	}
	if got != 15 {
		t.Fatalf("effective after higher id = %d, want 15", got)
	}
}

func TestReadStateRepository_GetReadStateForAllMembers(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	readRepo := NewReadStateRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice-rs")
	bobID := createTestUser(t, db, "bob-rs")

	chat, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}

	if _, err := readRepo.UpsertReadState(ctx, chat.ID, aliceID, 7); err != nil {
		t.Fatalf("UpsertReadState: %v", err)
	}

	states, err := readRepo.GetReadState(ctx, chat.ID)
	if err != nil {
		t.Fatalf("GetReadState: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("len(states) = %d, want 2", len(states))
	}

	byUser := map[int64]int64{}
	for _, s := range states {
		byUser[s.UserID] = s.LastReadMessageID
	}
	if byUser[aliceID] != 7 {
		t.Fatalf("alice cursor = %d, want 7", byUser[aliceID])
	}
	if byUser[bobID] != 0 {
		t.Fatalf("bob cursor = %d, want 0", byUser[bobID])
	}
}

func TestChatRepository_ListByUserIncludesReadCursor(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	msgRepo := NewMessageRepository(db)
	readRepo := NewReadStateRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice-list-read")
	bobID := createTestUser(t, db, "bob-list-read")

	chat, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}

	msg, err := msgRepo.Create(ctx, &domain.Message{
		ChatID:      chat.ID,
		SenderID:    bobID,
		ClientMsgID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Body:        "hi",
	})
	if err != nil {
		t.Fatalf("Create message: %v", err)
	}

	if _, err := readRepo.UpsertReadState(ctx, chat.ID, aliceID, msg.ID); err != nil {
		t.Fatalf("UpsertReadState: %v", err)
	}

	items, err := chatRepo.ListByUser(ctx, aliceID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].LastMessageID == nil || *items[0].LastMessageID != msg.ID {
		t.Fatalf("last_message_id = %v, want %d", items[0].LastMessageID, msg.ID)
	}
	if items[0].MyLastReadMessageID != msg.ID {
		t.Fatalf("my_last_read_message_id = %d, want %d", items[0].MyLastReadMessageID, msg.ID)
	}
}
