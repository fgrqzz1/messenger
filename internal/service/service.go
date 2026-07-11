package service

import (
	"messenger/internal/domain"
	"messenger/pkg/jwt"
)

type Service struct {
	users    domain.UserRepository
	chats    domain.ChatRepository
	messages domain.MessageRepository
	members  domain.MemberRepository
	jwt      *jwt.Manager
}

func New(
	users domain.UserRepository,
	chats domain.ChatRepository,
	messages domain.MessageRepository,
	members domain.MemberRepository,
	jwtManager *jwt.Manager,
) *Service {
	return &Service{
		users:    users,
		chats:    chats,
		messages: messages,
		members:  members,
		jwt:      jwtManager,
	}
}
