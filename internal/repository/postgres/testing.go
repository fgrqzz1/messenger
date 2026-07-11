package postgres

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupTestDB(t *testing.T) *DB {
	t.Helper()
	return &DB{pool: setupTestPool(t)}
}

func setupTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("messenger_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	if err := applyMigrations(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return pool
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return os.ErrInvalid
	}

	migrationPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "migrations", "000001_init_schema.up.sql")
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, string(sqlBytes))
	return err
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	return SetupTestDB(t)
}

func createTestUser(t *testing.T, db *DB, login string) int64 {
	t.Helper()

	repo := NewUserRepository(db)
	user, err := repo.Create(context.Background(), login, "hash")
	if err != nil {
		t.Fatalf("create user %q: %v", login, err)
	}

	return user.ID
}
