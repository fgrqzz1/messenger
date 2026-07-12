package postgres

import (
	"context"
	"fmt"

	"messenger/internal/domain"
)

type ChatRepository struct {
	db *DB
}

func (r *ChatRepository) CreateDirect(ctx context.Context, userAID, userBID int64) (*domain.Chat, error) {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertChat = `
		INSERT INTO chats (type, user_a_id, user_b_id)
		VALUES ('direct', $1, $2)
		RETURNING id, type, title, user_a_id, user_b_id, created_by, created_at
	`

	var chat domain.Chat
	var chatType string
	err = tx.QueryRow(ctx, insertChat, userAID, userBID).Scan(
		&chat.ID,
		&chatType,
		&chat.Title,
		&chat.UserAID,
		&chat.UserBID,
		&chat.CreatedBy,
		&chat.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	chat.Type = domain.ChatType(chatType)

	const insertMember = `
		INSERT INTO chat_members (chat_id, user_id, role)
		VALUES ($1, $2, 'member')
	`

	for _, userID := range []int64{userAID, userBID} {
		if _, err := tx.Exec(ctx, insertMember, chat.ID, userID); err != nil {
			return nil, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("postgres: commit tx: %w", err)
	}

	return &chat, nil
}

func (r *ChatRepository) CreateGroup(ctx context.Context, title string, createdBy int64) (*domain.Chat, error) {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertChat = `
		INSERT INTO chats (type, title, created_by)
		VALUES ('group', $1, $2)
		RETURNING id, type, title, user_a_id, user_b_id, created_by, created_at
	`

	var chat domain.Chat
	var chatType string
	err = tx.QueryRow(ctx, insertChat, title, createdBy).Scan(
		&chat.ID,
		&chatType,
		&chat.Title,
		&chat.UserAID,
		&chat.UserBID,
		&chat.CreatedBy,
		&chat.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	chat.Type = domain.ChatType(chatType)

	const insertMember = `
		INSERT INTO chat_members (chat_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`
	if _, err := tx.Exec(ctx, insertMember, chat.ID, createdBy); err != nil {
		return nil, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("postgres: commit tx: %w", err)
	}

	return &chat, nil
}

func (r *ChatRepository) GetDirectByUsers(ctx context.Context, userAID, userBID int64) (*domain.Chat, error) {
	const q = `
		SELECT id, type, title, user_a_id, user_b_id, created_by, created_at
		FROM chats
		WHERE type = 'direct'
		  AND LEAST(user_a_id, user_b_id) = LEAST($1::bigint, $2::bigint)
		  AND GREATEST(user_a_id, user_b_id) = GREATEST($1::bigint, $2::bigint)
	`

	var chat domain.Chat
	var chatType string
	err := r.db.pool.QueryRow(ctx, q, userAID, userBID).Scan(
		&chat.ID,
		&chatType,
		&chat.Title,
		&chat.UserAID,
		&chat.UserBID,
		&chat.CreatedBy,
		&chat.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	chat.Type = domain.ChatType(chatType)

	return &chat, nil
}

func (r *ChatRepository) GetByID(ctx context.Context, id int64) (*domain.Chat, error) {
	const q = `
		SELECT id, type, title, user_a_id, user_b_id, created_by, created_at
		FROM chats
		WHERE id = $1
	`

	var chat domain.Chat
	var chatType string
	err := r.db.pool.QueryRow(ctx, q, id).Scan(
		&chat.ID,
		&chatType,
		&chat.Title,
		&chat.UserAID,
		&chat.UserBID,
		&chat.CreatedBy,
		&chat.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	chat.Type = domain.ChatType(chatType)

	return &chat, nil
}

func (r *ChatRepository) ListByUser(ctx context.Context, userID int64) ([]domain.ChatListItem, error) {
	const q = `
		SELECT c.id, c.type, c.title, lm.id, lm.body, lm.created_at,
		       COALESCE(crs.last_read_message_id, 0)
		FROM chats c
		JOIN chat_members cm ON cm.chat_id = c.id AND cm.user_id = $1
		LEFT JOIN LATERAL (
		    SELECT id, body, created_at
		    FROM messages m
		    WHERE m.chat_id = c.id
		    ORDER BY m.id DESC
		    LIMIT 1
		) lm ON true
		LEFT JOIN chat_read_state crs ON crs.chat_id = c.id AND crs.user_id = $1
		ORDER BY lm.created_at DESC NULLS LAST
	`

	rows, err := r.db.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	defer rows.Close()

	var items []domain.ChatListItem
	for rows.Next() {
		var item domain.ChatListItem
		var chatType string
		if err := rows.Scan(
			&item.ID,
			&chatType,
			&item.Title,
			&item.LastMessageID,
			&item.LastMessageBody,
			&item.LastMessageAt,
			&item.MyLastReadMessageID,
		); err != nil {
			return nil, fmt.Errorf("postgres: scan chat list item: %w", err)
		}
		item.Type = domain.ChatType(chatType)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	if items == nil {
		items = []domain.ChatListItem{}
	}

	return items, nil
}

var _ domain.ChatRepository = (*ChatRepository)(nil)
