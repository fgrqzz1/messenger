package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOriginAllowAllWhenEmpty(t *testing.T) {
	check := newCheckOrigin(nil)

	req := httptest.NewRequest(http.MethodGet, "http://192.168.1.10/ws", nil)
	req.Header.Set("Origin", "http://192.168.1.10")
	if !check(req) {
		t.Fatal("empty allow-list must accept any Origin (dev/LAN trade-off)")
	}
}

func TestCheckOriginExactList(t *testing.T) {
	check := newCheckOrigin([]string{
		"http://localhost",
		"http://localhost:5173",
		"http://192.168.1.10",
	})

	cases := []struct {
		origin string
		want   bool
	}{
		{"http://localhost", true},
		{"http://localhost:5173", true},
		{"http://192.168.1.10", true},
		{"http://192.168.1.10:80", false},
		{"http://192.168.1.11", false},
		{"https://localhost", false},
		{"", true}, // нет Origin — пропускаем
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://backend/ws", nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		if got := check(req); got != tc.want {
			t.Fatalf("Origin %q: got %v, want %v", tc.origin, got, tc.want)
		}
	}
}

func TestNormalizeOrigin(t *testing.T) {
	got, ok := normalizeOrigin("HTTP://LocalHost:5173/")
	if !ok || got != "http://localhost:5173" {
		t.Fatalf("normalizeOrigin = %q, %v", got, ok)
	}
}
