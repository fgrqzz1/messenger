package service

import (
	"context"
	"strings"

	"messenger/internal/domain"
	"messenger/pkg/password"
)

const (
	defaultUserSearchLimit = 20
	maxUserSearchLimit     = 50
	minUserSearchQueryLen  = 2
)

func validateLogin(login string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return "", domain.ErrValidation
	}
	return login, nil
}

func validatePassword(raw string) error {
	if raw == "" {
		return domain.ErrValidation
	}
	return nil
}

func (s *Service) GetMe(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *Service) UpdateLogin(ctx context.Context, userID int64, login string) (*domain.User, error) {
	login, err := validateLogin(login)
	if err != nil {
		return nil, err
	}

	user, err := s.users.UpdateLogin(ctx, userID, login)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *Service) UpdatePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	if err := validatePassword(currentPassword); err != nil {
		return err
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	ok, err := password.Verify(currentPassword, user.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrInvalidCredentials
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}

	return s.users.UpdatePasswordHash(ctx, userID, hash)
}

func (s *Service) SearchUsers(ctx context.Context, callerID int64, login string, limit int) ([]domain.User, error) {
	login = strings.TrimSpace(login)
	if len([]rune(login)) < minUserSearchQueryLen {
		return nil, domain.ErrValidation
	}

	if limit <= 0 {
		limit = defaultUserSearchLimit
	}
	if limit > maxUserSearchLimit {
		limit = maxUserSearchLimit
	}

	return s.users.SearchByLogin(ctx, login, callerID, limit)
}
