package http

import (
	"net/http"

	"messenger/internal/service"
	"messenger/pkg/jwt"
)

func NewMux(svc *service.Service, jwtManager *jwt.Manager) http.Handler {
	h := NewHandler(svc, jwtManager)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /refresh", h.Refresh)

	withAuth := func(next http.Handler) http.Handler {
		return Auth(jwtManager, next)
	}

	mux.Handle("GET /chats", withAuth(http.HandlerFunc(h.ListChats)))
	mux.Handle("POST /chats", withAuth(http.HandlerFunc(h.CreateChat)))
	mux.Handle("GET /chats/{id}/messages", withAuth(http.HandlerFunc(h.ListMessages)))
	mux.Handle("GET /chats/{id}/members", withAuth(http.HandlerFunc(h.ListMembers)))
	mux.Handle("POST /chats/{id}/members", withAuth(http.HandlerFunc(h.AddMember)))
	mux.Handle("DELETE /chats/{id}/members/{user_id}", withAuth(http.HandlerFunc(h.RemoveMember)))
	mux.Handle("GET /chats/{id}/search", withAuth(http.HandlerFunc(h.SearchMessages)))

	return mux
}
