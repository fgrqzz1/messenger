package domain

import "time"

type ChatType string

const (
	ChatTypeDirect ChatType = "direct"
	ChatTypeGroup  ChatType = "group"
)

type MemberRole string

const (
	RoleMember MemberRole = "member"
	RoleAdmin  MemberRole = "admin"
)

type Chat struct {
	ID        int64
	Type      ChatType
	Title     *string
	UserAID   *int64
	UserBID   *int64
	CreatedBy *int64
	CreatedAt time.Time
}

type ChatMember struct {
	ChatID   int64
	UserID   int64
	Login    string
	Role     MemberRole
	JoinedAt time.Time
}

type ChatListItem struct {
	ID                  int64
	Type                ChatType
	Title               *string
	LastMessageID       *int64
	LastMessageBody     *string
	LastMessageAt       *time.Time
	MyLastReadMessageID int64
}
