package service

import (
	"context"

	"messenger/internal/domain"
)

type mockUserRepo struct {
	createFn     func(ctx context.Context, login, passwordHash string) (*domain.User, error)
	getByLoginFn func(ctx context.Context, login string) (*domain.User, error)
	getByIDFn    func(ctx context.Context, id int64) (*domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, login, passwordHash string) (*domain.User, error) {
	return m.createFn(ctx, login, passwordHash)
}

func (m *mockUserRepo) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	return m.getByLoginFn(ctx, login)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return m.getByIDFn(ctx, id)
}

type mockChatRepo struct {
	createDirectFn     func(ctx context.Context, userAID, userBID int64) (*domain.Chat, error)
	createGroupFn      func(ctx context.Context, title string, createdBy int64) (*domain.Chat, error)
	getByIDFn          func(ctx context.Context, id int64) (*domain.Chat, error)
	getDirectByUsersFn func(ctx context.Context, userAID, userBID int64) (*domain.Chat, error)
	listByUserFn       func(ctx context.Context, userID int64) ([]domain.ChatListItem, error)
}

func (m *mockChatRepo) CreateDirect(ctx context.Context, userAID, userBID int64) (*domain.Chat, error) {
	return m.createDirectFn(ctx, userAID, userBID)
}

func (m *mockChatRepo) CreateGroup(ctx context.Context, title string, createdBy int64) (*domain.Chat, error) {
	return m.createGroupFn(ctx, title, createdBy)
}

func (m *mockChatRepo) GetByID(ctx context.Context, id int64) (*domain.Chat, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockChatRepo) GetDirectByUsers(ctx context.Context, userAID, userBID int64) (*domain.Chat, error) {
	return m.getDirectByUsersFn(ctx, userAID, userBID)
}

func (m *mockChatRepo) ListByUser(ctx context.Context, userID int64) ([]domain.ChatListItem, error) {
	return m.listByUserFn(ctx, userID)
}

type mockMessageRepo struct {
	createFn     func(ctx context.Context, msg *domain.Message) (*domain.Message, error)
	listByChatFn func(ctx context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error)
	searchFn     func(ctx context.Context, chatID int64, query string) ([]domain.Message, error)
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	return m.createFn(ctx, msg)
}

func (m *mockMessageRepo) ListByChat(ctx context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error) {
	return m.listByChatFn(ctx, chatID, beforeID, limit)
}

func (m *mockMessageRepo) Search(ctx context.Context, chatID int64, query string) ([]domain.Message, error) {
	return m.searchFn(ctx, chatID, query)
}

type mockMemberRepo struct {
	addFn          func(ctx context.Context, member *domain.ChatMember) error
	removeFn       func(ctx context.Context, chatID, userID int64) error
	getFn          func(ctx context.Context, chatID, userID int64) (*domain.ChatMember, error)
	listUserIDsFn  func(ctx context.Context, chatID int64) ([]int64, error)
	listByChatFn   func(ctx context.Context, chatID int64) ([]domain.ChatMember, error)
}

func (m *mockMemberRepo) Add(ctx context.Context, member *domain.ChatMember) error {
	return m.addFn(ctx, member)
}

func (m *mockMemberRepo) Remove(ctx context.Context, chatID, userID int64) error {
	return m.removeFn(ctx, chatID, userID)
}

func (m *mockMemberRepo) Get(ctx context.Context, chatID, userID int64) (*domain.ChatMember, error) {
	return m.getFn(ctx, chatID, userID)
}

func (m *mockMemberRepo) ListUserIDs(ctx context.Context, chatID int64) ([]int64, error) {
	if m.listUserIDsFn != nil {
		return m.listUserIDsFn(ctx, chatID)
	}
	return nil, nil
}

func (m *mockMemberRepo) ListByChat(ctx context.Context, chatID int64) ([]domain.ChatMember, error) {
	if m.listByChatFn != nil {
		return m.listByChatFn(ctx, chatID)
	}
	return nil, nil
}
