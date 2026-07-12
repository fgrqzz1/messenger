package service

import (
	"context"
	"errors"

	"messenger/internal/domain"
	"messenger/pkg/jwt"
	"messenger/pkg/password"
)

func (s *Service) Register(ctx context.Context, login, rawPassword string) (*domain.User, error) {
	login, err := validateLogin(login)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(rawPassword); err != nil {
		return nil, err
	}

	hash, err := password.Hash(rawPassword)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(ctx, login, hash)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return user, nil
}

func (s *Service) Login(ctx context.Context, login, rawPassword string) (accessToken, refreshToken string, err error) {
	login, err = validateLogin(login)
	if err != nil {
		return "", "", err
	}
	if err := validatePassword(rawPassword); err != nil {
		return "", "", err
	}

	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", err
	}

	ok, err := password.Verify(rawPassword, user.PasswordHash)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", domain.ErrInvalidCredentials
	}

	pair, err := s.jwt.IssuePair(user.ID)
	if err != nil {
		return "", "", err
	}

	return pair.AccessToken, pair.RefreshToken, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (accessToken string, err error) {
	if refreshToken == "" {
		return "", domain.ErrUnauthorized
	}

	userID, err := s.jwt.ParseRefresh(refreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrInvalidToken) {
			return "", domain.ErrUnauthorized
		}
		return "", err
	}

	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return "", err
	}

	accessToken, err = s.jwt.IssueAccess(userID)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}
