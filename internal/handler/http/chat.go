package http

import (
	"net/http"

	"messenger/internal/domain"
)

type chatListItemResponse struct {
	ID                  int64   `json:"id"`
	Type                string  `json:"type"`
	Title               *string `json:"title,omitempty"`
	LastMessageID       *int64  `json:"last_message_id,omitempty"`
	LastMessageBody     *string `json:"last_message_body,omitempty"`
	LastMessageAt       *string `json:"last_message_at,omitempty"`
	MyLastReadMessageID int64   `json:"my_last_read_message_id"`
}

func (h *Handler) ListChats(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	items, err := h.svc.GetChats(r.Context(), callerID)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := make([]chatListItemResponse, 0, len(items))
	for _, item := range items {
		var lastAt *string
		if item.LastMessageAt != nil {
			formatted := item.LastMessageAt.UTC().Format(timeRFC3339Nano)
			lastAt = &formatted
		}
		resp = append(resp, chatListItemResponse{
			ID:                  item.ID,
			Type:                string(item.Type),
			Title:               item.Title,
			LastMessageID:       item.LastMessageID,
			LastMessageBody:     item.LastMessageBody,
			LastMessageAt:       lastAt,
			MyLastReadMessageID: item.MyLastReadMessageID,
		})
	}

	h.writeJSON(w, http.StatusOK, resp)
}

type createChatRequest struct {
	Type   string `json:"type"`
	Title  string `json:"title,omitempty"`
	UserID int64  `json:"user_id,omitempty"`
}

type chatResponse struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	Title     *string `json:"title,omitempty"`
	UserAID   *int64  `json:"user_a_id,omitempty"`
	UserBID   *int64  `json:"user_b_id,omitempty"`
	CreatedBy *int64  `json:"created_by,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func (h *Handler) CreateChat(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	var req createChatRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	var chat *domain.Chat
	var err error

	switch req.Type {
	case string(domain.ChatTypeDirect):
		if req.UserID <= 0 {
			writeError(w, domain.ErrValidation)
			return
		}
		chat, err = h.svc.CreateDirectChat(r.Context(), callerID, req.UserID)
	case string(domain.ChatTypeGroup):
		chat, err = h.svc.CreateGroupChat(r.Context(), callerID, req.Title)
	default:
		writeError(w, domain.ErrValidation)
		return
	}

	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toChatResponse(chat))
}

type updateChatRequest struct {
	Title string `json:"title"`
}

func (h *Handler) UpdateChat(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	var req updateChatRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	chat, err := h.svc.UpdateChatTitle(r.Context(), callerID, chatID, req.Title)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toChatResponse(chat))
}

func toChatResponse(chat *domain.Chat) chatResponse {
	return chatResponse{
		ID:        chat.ID,
		Type:      string(chat.Type),
		Title:     chat.Title,
		UserAID:   chat.UserAID,
		UserBID:   chat.UserBID,
		CreatedBy: chat.CreatedBy,
		CreatedAt: chat.CreatedAt.UTC().Format(timeRFC3339Nano),
	}
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
