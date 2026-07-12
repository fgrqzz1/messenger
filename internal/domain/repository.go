package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	SearchByLogin(ctx context.Context, query string, excludeUserID int64, limit int) ([]User, error)
}

type ChatRepository interface {
	CreateDirect(ctx context.Context, userAID, userBID int64) (*Chat, error)
	CreateGroup(ctx context.Context, title string, createdBy int64) (*Chat, error)
	GetByID(ctx context.Context, id int64) (*Chat, error)
	GetDirectByUsers(ctx context.Context, userAID, userBID int64) (*Chat, error)
	ListByUser(ctx context.Context, userID int64) ([]ChatListItem, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) (*Message, error)
	ListByChat(ctx context.Context, chatID, beforeID int64, limit int) ([]Message, error)
	Search(ctx context.Context, chatID int64, query string) ([]Message, error)
}

type MemberRepository interface {
	Add(ctx context.Context, member *ChatMember) error
	Remove(ctx context.Context, chatID, userID int64) error
	Get(ctx context.Context, chatID, userID int64) (*ChatMember, error)
	ListUserIDs(ctx context.Context, chatID int64) ([]int64, error)
	ListByChat(ctx context.Context, chatID int64) ([]ChatMember, error)
}

type ReadStateRepository interface {
	UpsertReadState(ctx context.Context, chatID, userID, messageID int64) (int64, error)
	GetReadState(ctx context.Context, chatID int64) ([]ChatReadState, error)
	IsReadByAll(ctx context.Context, chatID, messageID, excludeUserID int64) (bool, error)
}
