package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

const (
	claimType       = "typ"
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken = errors.New("jwt: invalid token")
)

type Config struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type Claims struct {
	UserID int64
	Type   string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type Manager struct {
	cfg Config
}

func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) IssuePair(userID int64) (*TokenPair, error) {
	access, err := m.issue(userID, tokenTypeAccess, m.cfg.AccessSecret, m.cfg.AccessTTL)
	if err != nil {
		return nil, err
	}

	refresh, err := m.issue(userID, tokenTypeRefresh, m.cfg.RefreshSecret, m.cfg.RefreshTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (m *Manager) IssueAccess(userID int64) (string, error) {
	return m.issue(userID, tokenTypeAccess, m.cfg.AccessSecret, m.cfg.AccessTTL)
}

func (m *Manager) ParseAccess(token string) (int64, error) {
	return m.parse(token, tokenTypeAccess, m.cfg.AccessSecret)
}

func (m *Manager) ParseRefresh(token string) (int64, error) {
	return m.parse(token, tokenTypeRefresh, m.cfg.RefreshSecret)
}

func (m *Manager) issue(userID int64, typ, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := gojwt.MapClaims{
		"sub":     strconv.FormatInt(userID, 10),
		claimType: typ,
		"iat":     now.Unix(),
		"exp":     now.Add(ttl).Unix(),
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("jwt: sign token: %w", err)
	}

	return signed, nil
}

func (m *Manager) parse(tokenString, expectedType, secret string) (int64, error) {
	token, err := gojwt.Parse(tokenString, func(token *gojwt.Token) (any, error) {
		if token.Method != gojwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}

	claims, ok := token.Claims.(gojwt.MapClaims)
	if !ok || !token.Valid {
		return 0, ErrInvalidToken
	}

	typ, _ := claims[claimType].(string)
	if typ != expectedType {
		return 0, ErrInvalidToken
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}

	return userID, nil
}
