package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"messenger/internal/domain"
)

func (s *Service) SendMessage(ctx context.Context, callerID, chatID int64, clientMsgID, body string) (*domain.Message, error) {
	clientMsgID = strings.TrimSpace(clientMsgID)
	body = strings.TrimSpace(body)
	if clientMsgID == "" || body == "" {
		return nil, domain.ErrValidation
	}
	if _, err := uuid.Parse(clientMsgID); err != nil {
		return nil, domain.ErrValidation
	}

	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	return s.messages.Create(ctx, &domain.Message{
		ChatID:      chatID,
		SenderID:    callerID,
		ClientMsgID: clientMsgID,
		Body:        body,
	})
}

func (s *Service) GetMessageHistory(ctx context.Context, callerID, chatID, beforeID int64, limit int) ([]domain.Message, error) {
	if limit <= 0 {
		return nil, domain.ErrValidation
	}

	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	return s.messages.ListByChat(ctx, chatID, beforeID, limit)
}
