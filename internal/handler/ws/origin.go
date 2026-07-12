package ws

import (
	"net/http"
	"net/url"
	"strings"
)

// newCheckOrigin возвращает CheckOrigin для gorilla/websocket.
//
// Пустой allowedOrigins — разрешены все Origin (осознанный trade-off для
// локальной разработки и доступа из LAN; не использовать так в production).
// Непустой список — точное совпадение Origin с одним из разрешённых значений
// (схема+хост[+порт], без path), например http://192.168.1.10 или http://localhost:5173.
func newCheckOrigin(allowedOrigins []string) func(*http.Request) bool {
	if len(allowedOrigins) == 0 {
		return func(_ *http.Request) bool { return true }
	}

	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, raw := range allowedOrigins {
		normalized, ok := normalizeOrigin(raw)
		if !ok {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Небраузерные клиенты / same-origin без Origin.
			return true
		}
		normalized, ok := normalizeOrigin(origin)
		if !ok {
			return false
		}
		_, ok = allowed[normalized]
		return ok
	}
}

func normalizeOrigin(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}

	return scheme + "://" + strings.ToLower(u.Host), true
}
