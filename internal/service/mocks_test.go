package service

import (
	"context"

	"messenger/internal/domain"
)

type mockUserRepo struct {
	createFn             func(ctx context.Context, login, passwordHash string) (*domain.User, error)
	getByLoginFn         func(ctx context.Context, login string) (*domain.User, error)
	getByIDFn            func(ctx context.Context, id int64) (*domain.User, error)
	searchByLoginFn      func(ctx context.Context, query string, excludeUserID int64, limit int) ([]domain.User, error)
	updateLoginFn        func(ctx context.Context, userID int64, login string) (*domain.User, error)
	updatePasswordHashFn func(ctx context.Context, userID int64, passwordHash string) error
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

func (m *mockUserRepo) SearchByLogin(ctx context.Context, query string, excludeUserID int64, limit int) ([]domain.User, error) {
	if m.searchByLoginFn != nil {
		return m.searchByLoginFn(ctx, query, excludeUserID, limit)
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateLogin(ctx context.Context, userID int64, login string) (*domain.User, error) {
	if m.updateLoginFn != nil {
		return m.updateLoginFn(ctx, userID, login)
	}
	return nil, nil
}

func (m *mockUserRepo) UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string) error {
	if m.updatePasswordHashFn != nil {
		return m.updatePasswordHashFn(ctx, userID, passwordHash)
	}
	return nil
}

type mockChatRepo struct {
	createDirectFn     func(ctx context.Context, userAID, userBID int64) (*domain.Chat, error)
	createGroupFn      func(ctx context.Context, title string, createdBy int64) (*domain.Chat, error)
	updateChatTitleFn  func(ctx context.Context, chatID int64, title string) (*domain.Chat, error)
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

func (m *mockChatRepo) UpdateChatTitle(ctx context.Context, chatID int64, title string) (*domain.Chat, error) {
	if m.updateChatTitleFn != nil {
		return m.updateChatTitleFn(ctx, chatID, title)
	}
	return nil, nil
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

type mockReadStateRepo struct {
	upsertFn      func(ctx context.Context, chatID, userID, messageID int64) (int64, error)
	getFn         func(ctx context.Context, chatID int64) ([]domain.ChatReadState, error)
	isReadByAllFn func(ctx context.Context, chatID, messageID, excludeUserID int64) (bool, error)
}

func (m *mockReadStateRepo) UpsertReadState(ctx context.Context, chatID, userID, messageID int64) (int64, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, chatID, userID, messageID)
	}
	return messageID, nil
}

func (m *mockReadStateRepo) GetReadState(ctx context.Context, chatID int64) ([]domain.ChatReadState, error) {
	if m.getFn != nil {
		return m.getFn(ctx, chatID)
	}
	return nil, nil
}

func (m *mockReadStateRepo) IsReadByAll(ctx context.Context, chatID, messageID, excludeUserID int64) (bool, error) {
	if m.isReadByAllFn != nil {
		return m.isReadByAllFn(ctx, chatID, messageID, excludeUserID)
	}
	return false, nil
}

type mockRealtimeNotifier struct {
	notifyReadFn        func(ctx context.Context, chatID, userID, lastReadMessageID int64)
	notifyChatUpdatedFn func(ctx context.Context, chatID, actorUserID int64, title string)
	calls               []notifyReadCall
	chatUpdatedCalls    []notifyChatUpdatedCall
}

type notifyReadCall struct {
	chatID            int64
	userID            int64
	lastReadMessageID int64
}

type notifyChatUpdatedCall struct {
	chatID      int64
	actorUserID int64
	title       string
}

func (m *mockRealtimeNotifier) NotifyRead(ctx context.Context, chatID, userID, lastReadMessageID int64) {
	m.calls = append(m.calls, notifyReadCall{
		chatID:            chatID,
		userID:            userID,
		lastReadMessageID: lastReadMessageID,
	})
	if m.notifyReadFn != nil {
		m.notifyReadFn(ctx, chatID, userID, lastReadMessageID)
	}
}

func (m *mockRealtimeNotifier) NotifyChatUpdated(ctx context.Context, chatID, actorUserID int64, title string) {
	m.chatUpdatedCalls = append(m.chatUpdatedCalls, notifyChatUpdatedCall{
		chatID:      chatID,
		actorUserID: actorUserID,
		title:       title,
	})
	if m.notifyChatUpdatedFn != nil {
		m.notifyChatUpdatedFn(ctx, chatID, actorUserID, title)
	}
}
