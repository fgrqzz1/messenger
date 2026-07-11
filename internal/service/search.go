package service

import (
	"context"
	"strings"

	"messenger/internal/domain"
)

func (s *Service) Search(ctx context.Context, callerID, chatID int64, query string) ([]domain.Message, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, domain.ErrValidation
	}

	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	return s.messages.Search(ctx, chatID, query)
}
