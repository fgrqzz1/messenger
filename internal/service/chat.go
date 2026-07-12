package service

import (
	"context"
	"errors"
	"strings"

	"messenger/internal/domain"
)

const maxChatTitleLen = 100

func validateChatTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", domain.ErrValidation
	}
	if len([]rune(title)) > maxChatTitleLen {
		return "", domain.ErrValidation
	}
	return title, nil
}

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
	title, err := validateChatTitle(title)
	if err != nil {
		return nil, err
	}

	return s.chats.CreateGroup(ctx, title, callerID)
}

func (s *Service) UpdateChatTitle(ctx context.Context, callerID, chatID int64, title string) (*domain.Chat, error) {
	title, err := validateChatTitle(title)
	if err != nil {
		return nil, err
	}

	if err := s.ensureGroupChatAdmin(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	chat, err := s.chats.UpdateChatTitle(ctx, chatID, title)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.NotifyChatUpdated(ctx, chatID, callerID, title)
	}

	return chat, nil
}

func (s *Service) GetChats(ctx context.Context, userID int64) ([]domain.ChatListItem, error) {
	return s.chats.ListByUser(ctx, userID)
}
