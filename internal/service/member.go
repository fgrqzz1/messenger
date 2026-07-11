package service

import (
	"context"
	"errors"

	"messenger/internal/domain"
)

func (s *Service) AddMember(ctx context.Context, callerID, chatID, userID int64) error {
	if err := s.ensureGroupChatAdmin(ctx, chatID, callerID); err != nil {
		return err
	}

	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return err
	}

	err := s.members.Add(ctx, &domain.ChatMember{
		ChatID: chatID,
		UserID: userID,
		Role:   domain.RoleMember,
	})
	if errors.Is(err, domain.ErrConflict) {
		return domain.ErrConflict
	}

	return err
}

func (s *Service) RemoveMember(ctx context.Context, callerID, chatID, userID int64) error {
	if err := s.ensureGroupChatAdmin(ctx, chatID, callerID); err != nil {
		return err
	}

	return s.members.Remove(ctx, chatID, userID)
}

func (s *Service) ListChatMemberUserIDs(ctx context.Context, chatID, callerID int64) ([]int64, error) {
	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	return s.members.ListUserIDs(ctx, chatID)
}

func (s *Service) ListMembers(ctx context.Context, callerID, chatID int64) ([]domain.ChatMember, error) {
	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	return s.members.ListByChat(ctx, chatID)
}

func (s *Service) ensureGroupChatAdmin(ctx context.Context, chatID, userID int64) error {
	chat, err := s.chats.GetByID(ctx, chatID)
	if err != nil {
		return err
	}
	if chat.Type != domain.ChatTypeGroup {
		return domain.ErrValidation
	}

	member, err := s.members.Get(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if member.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}

	return nil
}

func (s *Service) ensureChatMember(ctx context.Context, chatID, userID int64) error {
	_, err := s.members.Get(ctx, chatID, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrForbidden
	}

	return err
}
