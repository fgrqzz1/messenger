package service

import (
	"context"
	"errors"
	"strings"

	"messenger/internal/domain"
)

func (s *Service) CreateDirectChat(ctx context.Context, callerID, otherUserID int64) (*domain.Chat, error) {
	if callerID == otherUserID {
		return nil, domain.ErrValidation
	}

	if _, err := s.users.GetByID(ctx, otherUserID); err != nil {
		return nil, err
	}

	chat, err := s.chats.CreateDirect(ctx, callerID, otherUserID)
	if err == nil {
		return chat, nil
	}
	if errors.Is(err, domain.ErrConflict) {
		return s.chats.GetDirectByUsers(ctx, callerID, otherUserID)
	}

	return nil, err
}

func (s *Service) CreateGroupChat(ctx context.Context, callerID int64, title string) (*domain.Chat, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, domain.ErrValidation
	}

	return s.chats.CreateGroup(ctx, title, callerID)
}

func (s *Service) GetChats(ctx context.Context, userID int64) ([]domain.ChatListItem, error) {
	return s.chats.ListByUser(ctx, userID)
}
