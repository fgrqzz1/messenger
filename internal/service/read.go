package service

import (
	"context"

	"messenger/internal/domain"
)

func (s *Service) MarkRead(ctx context.Context, chatID, userID, messageID int64) error {
	if messageID < 0 {
		return domain.ErrValidation
	}
	if err := s.ensureChatMember(ctx, chatID, userID); err != nil {
		return err
	}

	effectiveID, err := s.readStates.UpsertReadState(ctx, chatID, userID, messageID)
	if err != nil {
		return err
	}

	if s.notifier != nil {
		s.notifier.NotifyRead(ctx, chatID, userID, effectiveID)
	}
	return nil
}

func (s *Service) GetReadState(ctx context.Context, chatID, userID int64) ([]domain.ChatReadState, error) {
	if err := s.ensureChatMember(ctx, chatID, userID); err != nil {
		return nil, err
	}
	return s.readStates.GetReadState(ctx, chatID)
}
