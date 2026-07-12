package postgres

import (
	"context"
	"fmt"

	"messenger/internal/domain"
)

type ReadStateRepository struct {
	db *DB
}

func (r *ReadStateRepository) UpsertReadState(ctx context.Context, chatID, userID, messageID int64) (int64, error) {
	const q = `
		INSERT INTO chat_read_state (chat_id, user_id, last_read_message_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, user_id) DO UPDATE SET
			last_read_message_id = GREATEST(
				chat_read_state.last_read_message_id,
				EXCLUDED.last_read_message_id
			),
			updated_at = now()
		RETURNING last_read_message_id
	`

	var effective int64
	err := r.db.pool.QueryRow(ctx, q, chatID, userID, messageID).Scan(&effective)
	if err != nil {
		return 0, mapError(err)
	}
	return effective, nil
}

func (r *ReadStateRepository) GetReadState(ctx context.Context, chatID int64) ([]domain.ChatReadState, error) {
	const q = `
		SELECT cm.user_id, COALESCE(crs.last_read_message_id, 0)
		FROM chat_members cm
		LEFT JOIN chat_read_state crs
			ON crs.chat_id = cm.chat_id AND crs.user_id = cm.user_id
		WHERE cm.chat_id = $1
		ORDER BY cm.user_id
	`

	rows, err := r.db.pool.Query(ctx, q, chatID)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	defer rows.Close()

	var states []domain.ChatReadState
	for rows.Next() {
		var s domain.ChatReadState
		s.ChatID = chatID
		if err := rows.Scan(&s.UserID, &s.LastReadMessageID); err != nil {
			return nil, fmt.Errorf("postgres: scan read state: %w", err)
		}
		states = append(states, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	if states == nil {
		states = []domain.ChatReadState{}
	}
	return states, nil
}

func (r *ReadStateRepository) IsReadByAll(ctx context.Context, chatID, messageID, excludeUserID int64) (bool, error) {
	const q = `
		SELECT NOT EXISTS (
			SELECT 1
			FROM chat_members cm
			LEFT JOIN chat_read_state crs
				ON crs.chat_id = cm.chat_id AND crs.user_id = cm.user_id
			WHERE cm.chat_id = $1
			  AND cm.user_id <> $2
			  AND COALESCE(crs.last_read_message_id, 0) < $3
		)
	`

	var ok bool
	if err := r.db.pool.QueryRow(ctx, q, chatID, excludeUserID, messageID).Scan(&ok); err != nil {
		return false, mapError(err)
	}
	return ok, nil
}

var _ domain.ReadStateRepository = (*ReadStateRepository)(nil)
