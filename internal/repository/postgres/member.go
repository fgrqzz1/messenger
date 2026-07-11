package postgres

import (
	"context"
	"fmt"

	"messenger/internal/domain"
)

type MemberRepository struct {
	db *DB
}

func (r *MemberRepository) Add(ctx context.Context, member *domain.ChatMember) error {
	const q = `
		INSERT INTO chat_members (chat_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING joined_at
	`

	err := r.db.pool.QueryRow(ctx, q, member.ChatID, member.UserID, string(member.Role)).Scan(&member.JoinedAt)
	if err != nil {
		return mapError(err)
	}

	return nil
}

func (r *MemberRepository) Remove(ctx context.Context, chatID, userID int64) error {
	const q = `
		DELETE FROM chat_members
		WHERE chat_id = $1 AND user_id = $2
	`

	tag, err := r.db.pool.Exec(ctx, q, chatID, userID)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *MemberRepository) Get(ctx context.Context, chatID, userID int64) (*domain.ChatMember, error) {
	const q = `
		SELECT chat_id, user_id, role, joined_at
		FROM chat_members
		WHERE chat_id = $1 AND user_id = $2
	`

	var member domain.ChatMember
	var role string
	err := r.db.pool.QueryRow(ctx, q, chatID, userID).Scan(
		&member.ChatID,
		&member.UserID,
		&role,
		&member.JoinedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	member.Role = domain.MemberRole(role)

	return &member, nil
}

func (r *MemberRepository) ListUserIDs(ctx context.Context, chatID int64) ([]int64, error) {
	const q = `
		SELECT user_id
		FROM chat_members
		WHERE chat_id = $1
		ORDER BY user_id
	`

	rows, err := r.db.pool.Query(ctx, q, chatID)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres: scan member user_id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	return ids, nil
}

func (r *MemberRepository) ListByChat(ctx context.Context, chatID int64) ([]domain.ChatMember, error) {
	const q = `
		SELECT cm.chat_id, cm.user_id, u.login, cm.role, cm.joined_at
		FROM chat_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.chat_id = $1
		ORDER BY cm.user_id
	`

	rows, err := r.db.pool.Query(ctx, q, chatID)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	defer rows.Close()

	var members []domain.ChatMember
	for rows.Next() {
		var member domain.ChatMember
		var role string
		if err := rows.Scan(
			&member.ChatID,
			&member.UserID,
			&member.Login,
			&role,
			&member.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres: scan chat member: %w", err)
		}
		member.Role = domain.MemberRole(role)
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	if members == nil {
		members = []domain.ChatMember{}
	}

	return members, nil
}

var _ domain.MemberRepository = (*MemberRepository)(nil)
