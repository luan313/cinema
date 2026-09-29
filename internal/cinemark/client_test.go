package cinemark

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func fixtureServer(t *testing.T) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == "" || r.Header.Get("Referer") == "" {
			http.Error(w, "blocked", http.StatusForbidden)
			return
		}
		files := map[string]string{
			"/v1/sessions/movieAndCity": "../../testdata/sessions.json",
			"/v1/seatmaps":              "../../testdata/seatmap.json",
		}
		f, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	c := New()
	c.BaseURL = srv.URL
	c.Retries = 0
	return c
}

func TestSessions(t *testing.T) {
	c := fixtureServer(t)
	days, err := c.Sessions(context.Background(), "9349", 9668)
	if err != nil {
		t.Fatal(err)
	}
	if len(days) == 0 || len(days[0].Rooms) == 0 || days[0].Rooms[0].Sessions[0].ID == "" {
		t.Fatalf("sessões não decodificadas: %+v", days)
	}
}

func TestSeatMap(t *testing.T) {
	c := fixtureServer(t)
	sm, err := c.SeatMap(context.Background(), 2120, "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(sm.Elements) < 100 {
		t.Fatalf("mapa pequeno demais: %d", len(sm.Elements))
	}
}
