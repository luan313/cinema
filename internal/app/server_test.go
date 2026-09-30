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

func TestAllowedTheaters(t *testing.T) {
	sameSet := func(got []int, want ...int) bool {
		if len(got) != len(want) {
			return false
		}
		m := map[int]bool{}
		for _, g := range got {
			m[g] = true
		}
		for _, w := range want {
			if !m[w] {
				return false
			}
		}
		return true
	}
	if got := allowedTheaters(nil); !sameSet(got, 2133, 2113) {
		t.Errorf("vazio: %v", got)
	}
	if got := allowedTheaters([]int{2133}); !sameSet(got, 2133) {
		t.Errorf("um cinema: %v", got)
	}
	if got := allowedTheaters([]int{999, 2113}); !sameSet(got, 2113) {
		t.Errorf("cinema de fora é ignorado: %v", got)
	}
	if got := allowedTheaters([]int{999}); !sameSet(got, 2133, 2113) {
		t.Errorf("só cinema de fora volta para os dois: %v", got)
	}
}
