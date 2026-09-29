package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostGuard(t *testing.T) {
	h := NewServer("test").Handler("127.0.0.1:9999")
	for _, tc := range []struct {
		host, origin string
		want         int
	}{
		{"127.0.0.1:9999", "", http.StatusOK},
		{"evil.example", "", http.StatusForbidden},
		{"127.0.0.1:9999", "http://evil.example", http.StatusForbidden},
	} {
		r := httptest.NewRequest("GET", "/api/version", nil)
		r.Host = tc.host
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("host=%s origin=%s: got %d want %d", tc.host, tc.origin, w.Code, tc.want)
		}
	}
}

func TestIndexServed(t *testing.T) {
	h := NewServer("test").Handler("127.0.0.1:9999")
	r := httptest.NewRequest("GET", "/", nil)
	r.Host = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.Len() < 500 {
		t.Fatalf("index não servido: %d", w.Code)
	}
}
