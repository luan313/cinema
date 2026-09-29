package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostGuard(t *testing.T) {
	h := NewServer("test").Handler(Config{Host: "127.0.0.1:9999"})
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
	h := NewServer("test").Handler(Config{Host: "127.0.0.1:9999"})
	r := httptest.NewRequest("GET", "/", nil)
	r.Host = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.Len() < 500 {
		t.Fatalf("index não servido: %d", w.Code)
	}
}

func TestRemoteMode(t *testing.T) {
	h := NewServer("test").Handler(Config{Password: "segredo"})
	do := func(host, origin, pass string) int {
		r := httptest.NewRequest("GET", "/api/version", nil)
		r.Host = host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if pass != "" {
			r.SetBasicAuth("x", pass)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	for _, tc := range []struct {
		name               string
		host, origin, pass string
		want               int
	}{
		{"sem senha", "app.example", "", "", http.StatusUnauthorized},
		{"senha errada", "app.example", "", "errada", http.StatusUnauthorized},
		{"ok", "app.example", "", "segredo", http.StatusOK},
		{"mesma origem https", "app.example", "https://app.example", "segredo", http.StatusOK},
		{"outra origem", "app.example", "https://evil.example", "segredo", http.StatusForbidden},
	} {
		if got := do(tc.host, tc.origin, tc.pass); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestIsLoopback(t *testing.T) {
	for addr, want := range map[string]bool{"127.0.0.1:80": true, "localhost:80": true, "[::1]:80": true, ":8080": false, "0.0.0.0:80": false, "10.0.0.5:80": false} {
		if IsLoopback(addr) != want {
			t.Errorf("%s: esperado %v", addr, want)
		}
	}
}

func TestRemoteModeForwardedHost(t *testing.T) {
	h := NewServer("test").Handler(Config{})
	r := httptest.NewRequest("POST", "/api/cancel", nil)
	r.Host = "localhost:8080"
	r.Header.Set("Origin", "https://abc-8080.app.github.dev")
	r.Header.Set("X-Forwarded-Host", "abc-8080.app.github.dev")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("com X-Forwarded-Host: %d", w.Code)
	}
	r.Header.Del("X-Forwarded-Host")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("sem X-Forwarded-Host deveria recusar: %d", w.Code)
	}
}
