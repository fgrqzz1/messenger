package http

import (
	"net/http"

	"messenger/internal/domain"
)

type meResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	CreatedAt string `json:"created_at"`
}

func meFromUser(u *domain.User) meResponse {
	return meResponse{
		ID:        u.ID,
		Login:     u.Login,
		CreatedAt: u.CreatedAt.UTC().Format(timeRFC3339Nano),
	}
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	user, err := h.svc.GetMe(r.Context(), callerID)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, meFromUser(user))
}

type updateLoginRequest struct {
	Login string `json:"login"`
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	var req updateLoginRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	user, err := h.svc.UpdateLogin(r.Context(), callerID, req.Login)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, meFromUser(user))
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	var req updatePasswordRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	if err := h.svc.UpdatePassword(r.Context(), callerID, req.CurrentPassword, req.NewPassword); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
