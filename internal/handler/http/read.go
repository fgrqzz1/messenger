package http

import (
	"net/http"

	"messenger/internal/domain"
)

type markReadRequest struct {
	LastReadMessageID int64 `json:"last_read_message_id"`
}

type readStateResponse struct {
	UserID            int64 `json:"user_id"`
	LastReadMessageID int64 `json:"last_read_message_id"`
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	var req markReadRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	if err := h.svc.MarkRead(r.Context(), chatID, callerID, req.LastReadMessageID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetReadState(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	states, err := h.svc.GetReadState(r.Context(), chatID, callerID)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := make([]readStateResponse, 0, len(states))
	for _, s := range states {
		resp = append(resp, readStateResponse{
			UserID:            s.UserID,
			LastReadMessageID: s.LastReadMessageID,
		})
	}

	h.writeJSON(w, http.StatusOK, resp)
}
