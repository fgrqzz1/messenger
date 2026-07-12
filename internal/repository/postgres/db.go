package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, connString string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func NewChatRepository(db *DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func NewMessageRepository(db *DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func NewMemberRepository(db *DB) *MemberRepository {
	return &MemberRepository{db: db}
}

func NewReadStateRepository(db *DB) *ReadStateRepository {
	return &ReadStateRepository{db: db}
}
