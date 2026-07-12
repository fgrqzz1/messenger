package postgres

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
)

func TestUserRepository_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, "alice", "hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.Login != "alice" {
		t.Fatalf("unexpected user: %+v", created)
	}

	byLogin, err := repo.GetByLogin(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if byLogin.ID != created.ID {
		t.Fatalf("GetByLogin id = %d, want %d", byLogin.ID, created.ID)
	}

	byID, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Login != "alice" {
		t.Fatalf("GetByID login = %q, want alice", byID.Login)
	}
}

func TestUserRepository_CreateDuplicateLogin(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	if _, err := repo.Create(ctx, "bob", "hash"); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err := repo.Create(ctx, "bob", "other")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second Create error = %v, want ErrConflict", err)
	}
}

func TestUserRepository_SearchByLogin(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	alice, err := repo.Create(ctx, "alice", "hash")
	if err != nil {
		t.Fatalf("Create alice: %v", err)
	}
	bob, err := repo.Create(ctx, "bobbie", "hash")
	if err != nil {
		t.Fatalf("Create bobbie: %v", err)
	}
	if _, err := repo.Create(ctx, "carol", "hash"); err != nil {
		t.Fatalf("Create carol: %v", err)
	}

	// Caller matching the query must be excluded.
	found, err := repo.SearchByLogin(ctx, "ali", alice.ID, 20)
	if err != nil {
		t.Fatalf("SearchByLogin: %v", err)
	}
	if len(found) != 0 {
		t.Fatalf("search as alice for \"ali\": got %+v, want empty (self excluded)", found)
	}

	found, err = repo.SearchByLogin(ctx, "bo", alice.ID, 20)
	if err != nil {
		t.Fatalf("SearchByLogin: %v", err)
	}
	if len(found) != 1 || found[0].ID != bob.ID || found[0].Login != "bobbie" {
		t.Fatalf("search for \"bo\": got %+v, want bobbie", found)
	}
	if found[0].PasswordHash != "" {
		t.Fatalf("PasswordHash leaked: %q", found[0].PasswordHash)
	}
}

func TestUserRepository_UpdateLogin(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	alice, err := repo.Create(ctx, "alice", "hash")
	if err != nil {
		t.Fatalf("Create alice: %v", err)
	}
	if _, err := repo.Create(ctx, "bob", "hash"); err != nil {
		t.Fatalf("Create bob: %v", err)
	}

	updated, err := repo.UpdateLogin(ctx, alice.ID, "alice2")
	if err != nil {
		t.Fatalf("UpdateLogin: %v", err)
	}
	if updated.Login != "alice2" || updated.PasswordHash != "" {
		t.Fatalf("unexpected user: %+v", updated)
	}

	_, err = repo.UpdateLogin(ctx, alice.ID, "bob")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("UpdateLogin conflict error = %v, want ErrConflict", err)
	}
}

func TestUserRepository_UpdatePasswordHash(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	alice, err := repo.Create(ctx, "alice", "old-hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdatePasswordHash(ctx, alice.ID, "new-hash"); err != nil {
		t.Fatalf("UpdatePasswordHash: %v", err)
	}

	got, err := repo.GetByID(ctx, alice.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.PasswordHash != "new-hash" {
		t.Fatalf("PasswordHash = %q, want new-hash", got.PasswordHash)
	}
}
