// Package app expõe a interface local (HTML embutido + API JSON) do executável.
package app

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/luan313/cinema/internal/analyze"
	"github.com/luan313/cinema/internal/cinemark"
)

//go:embed web
var webFS embed.FS

type job struct {
	mu       sync.Mutex
	Done     int           `json:"done"`
	Total    int           `json:"total"`
	Finished bool          `json:"finished"`
	Error    string        `json:"error,omitempty"`
	Rows     []analyze.Row `json:"rows"`
	cancel   context.CancelFunc
}

type Server struct {
	Client   *cinemark.Client
	Version  string
	mu       sync.Mutex
	jobs     map[string]*job
	lastPing time.Time
	pinged   bool
}

func NewServer(version string) *Server {
	return &Server{Client: cinemark.New(), Version: version, jobs: map[string]*job{}}
}

// Handler devolve as rotas; host é o "127.0.0.1:porta" aceito (anti DNS rebinding).
func (s *Server) Handler(host string) http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/ping", s.ping)
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, s.Version) })
	mux.HandleFunc("/api/states", s.proxy(func(ctx context.Context, r *http.Request) (any, error) { return s.Client.States(ctx) }))
	mux.HandleFunc("/api/cities", s.proxy(func(ctx context.Context, r *http.Request) (any, error) {
		return s.Client.Cities(ctx, intParam(r, "stateId"))
	}))
	mux.HandleFunc("/api/movies", s.proxy(func(ctx context.Context, r *http.Request) (any, error) {
		return s.Client.Movies(ctx, intParam(r, "cityId"))
	}))
	mux.HandleFunc("/api/options", s.proxy(s.options))
	mux.HandleFunc("/api/rows", s.proxy(s.rows))
	mux.HandleFunc("/api/search", s.search)
	mux.HandleFunc("/api/job", s.jobStatus)
	mux.HandleFunc("/api/cancel", s.cancel)
	return hostGuard(host, mux)
}

func hostGuard(host string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host {
			http.Error(w, "host inválido", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+host {
			http.Error(w, "origem inválida", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func intParam(r *http.Request, k string) int {
	n, _ := strconv.Atoi(r.URL.Query().Get(k))
	return n
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (s *Server) proxy(fn func(context.Context, *http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := fn(r.Context(), r)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, v)
	}
}

// options lista, para o filme escolhido, o que existe nas sessões (datas, cinemas...).
func (s *Server) options(ctx context.Context, r *http.Request) (any, error) {
	days, err := s.Client.Sessions(ctx, r.URL.Query().Get("movieId"), intParam(r, "cityId"))
	if err != nil {
		return nil, err
	}
	type named struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	dates := map[string]bool{}
	theaters := map[int]string{}
	feats, auds := map[int]bool{}, map[int]bool{}
	for _, d := range days {
		for _, room := range d.Rooms {
			live := false
			for _, ss := range room.Sessions {
				if !ss.Expired && len(ss.Date) >= 10 {
					dates[ss.Date[:10]] = true
					live = true
				}
			}
			if live {
				theaters[d.TheaterID] = d.TheaterName
				for _, f := range room.Features {
					feats[f] = true
				}
				auds[room.Audio] = true
			}
		}
	}
	out := struct {
		Dates    []string  `json:"dates"`
		Theaters []named   `json:"theaters"`
		Features []feature `json:"features"`
		Audios   []named   `json:"audios"`
	}{Dates: []string{}, Theaters: []named{}, Features: []feature{}, Audios: []named{}}
	for d := range dates {
		out.Dates = append(out.Dates, d)
	}
	sort.Strings(out.Dates)
	for id, n := range theaters {
		out.Theaters = append(out.Theaters, named{id, n})
	}
	sort.Slice(out.Theaters, func(i, j int) bool { return out.Theaters[i].Name < out.Theaters[j].Name })
	// Lista todos os formatos conhecidos (ex.: Infinity Vision), mesmo sem sessão
	// para este filme, para o filtro estar sempre disponível.
	for id, n := range analyze.FeatureNames {
		out.Features = append(out.Features, feature{ID: id, Name: n, Available: feats[id]})
	}
	sort.Slice(out.Features, func(i, j int) bool { return out.Features[i].ID < out.Features[j].ID })
	for id := range auds {
		if n, ok := analyze.AudioNames[id]; ok {
			out.Audios = append(out.Audios, named{id, n})
		}
	}
	sort.Slice(out.Audios, func(i, j int) bool { return out.Audios[i].ID < out.Audios[j].ID })
	return out, nil
}

// rows lista as fileiras das salas que exibem o filme, da mais perto da tela
// para a mais longe. Consulta o mapa de uma sessão por sala (no máximo 40).
func (s *Server) rows(ctx context.Context, r *http.Request) (any, error) {
	days, err := s.Client.Sessions(ctx, r.URL.Query().Get("movieId"), intParam(r, "cityId"))
	if err != nil {
		return nil, err
	}
	type room struct {
		theater int
		number  int
	}
	seen := map[room]bool{}
	type pick struct {
		theater int
		session string
	}
	var picks []pick
	for _, d := range days {
		for _, rm := range d.Rooms {
			k := room{d.TheaterID, rm.Number}
			if seen[k] || len(picks) >= 40 {
				continue
			}
			for _, ss := range rm.Sessions {
				if !ss.Expired {
					seen[k] = true
					picks = append(picks, pick{d.TheaterID, ss.ID})
					break
				}
			}
		}
	}
	maps := make([]*cinemark.SeatMap, len(picks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, p := range picks {
		wg.Add(1)
		go func(i int, p pick) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if sm, err := s.Client.SeatMap(ctx, p.theater, p.session); err == nil {
				maps[i] = sm
			}
		}(i, p)
	}
	wg.Wait()
	var ok []*cinemark.SeatMap
	for _, m := range maps {
		if m != nil {
			ok = append(ok, m)
		}
	}
	return analyze.RowOrder(ok), nil
}

type feature struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"` // há sessões deste formato para o filme
}

type searchReq struct {
	MovieID string          `json:"movieId"`
	CityID  int             `json:"cityId"`
	Filters analyze.Filters `json:"filters"`
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var req searchReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil || req.MovieID == "" || req.CityID == 0 {
		http.Error(w, "requisição inválida", http.StatusBadRequest)
		return
	}
	idb := make([]byte, 8)
	rand.Read(idb)
	id := hex.EncodeToString(idb)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	j := &job{cancel: cancel, Rows: []analyze.Row{}}
	s.mu.Lock()
	s.jobs[id] = j
	s.mu.Unlock()
	go func() {
		defer cancel()
		rows, err := analyze.Search(ctx, s.Client, req.MovieID, req.CityID, req.Filters, func(done, total int) {
			j.mu.Lock()
			j.Done, j.Total = done, total
			j.mu.Unlock()
		})
		j.mu.Lock()
		defer j.mu.Unlock()
		if err != nil {
			j.Error = err.Error()
		} else if rows != nil {
			j.Rows = rows
		}
		j.Finished = true
	}()
	writeJSON(w, map[string]string{"jobId": id})
}

func (s *Server) getJob(r *http.Request) *job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[r.URL.Query().Get("id")]
}

func (s *Server) jobStatus(w http.ResponseWriter, r *http.Request) {
	j := s.getJob(r)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	writeJSON(w, j)
}

func (s *Server) cancel(w http.ResponseWriter, r *http.Request) {
	if j := s.getJob(r); j != nil {
		j.cancel()
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.lastPing, s.pinged = time.Now(), true
	s.mu.Unlock()
	writeJSON(w, map[string]bool{"ok": true})
}

// WaitIdle bloqueia até o navegador parar de enviar ping (janela fechada).
func (s *Server) WaitIdle(grace time.Duration) {
	start := time.Now()
	for {
		time.Sleep(2 * time.Second)
		s.mu.Lock()
		pinged, last := s.pinged, s.lastPing
		s.mu.Unlock()
		if !pinged && time.Since(start) > 2*time.Minute {
			return
		}
		if pinged && time.Since(last) > grace {
			return
		}
	}
}

// Listen abre uma porta local aleatória e devolve o endereço.
func Listen() (net.Listener, string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	return l, fmt.Sprintf("127.0.0.1:%d", l.Addr().(*net.TCPAddr).Port), nil
}
