package analyze

import (
	"context"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/luan313/cinema/internal/cinemark"
)

// Nomes usados pelo site para os códigos de tecnologia e áudio.
var FeatureNames = map[int]string{1: "D-BOX", 2: "XD", 3: "IMAX", 4: "Ingresso Azul", 5: "Prime", 6: "3D", 7: "2D", 9: "Cine Materna", 10: "Infinity Vision"}
var AudioNames = map[int]string{10: "Original", 20: "Dublado", 30: "Legendado"}

type Filters struct {
	Dates          []string `json:"dates"`      // YYYY-MM-DD; vazio = todas
	TimeFrom       string   `json:"timeFrom"`   // HH:MM; vazio = sem limite
	TimeTo         string   `json:"timeTo"`     // HH:MM; vazio = sem limite
	TheaterIDs     []int    `json:"theaterIds"` // vazio = todos
	Features       []int    `json:"features"`   // vazio = todas
	Audios         []int    `json:"audios"`     // vazio = todos
	GroupSize      int      `json:"groupSize"`  // N assentos lado a lado
	MinFree        int      `json:"minFree"`    // mínimo de assentos livres
	OnlyWithGroups bool     `json:"onlyWithGroups"`
	IncludeSpecial bool     `json:"includeSpecial"`
	Vertical       bool     `json:"vertical"` // também vale assento na fileira da frente/de trás
	RowFrom        int      `json:"rowFrom"`  // 1ª fileira (1 = mais perto da tela); 0 = sem limite
	RowTo          int      `json:"rowTo"`    // última fileira; 0 = até a última da sala
}

type Row struct {
	TheaterID   int       `json:"theaterId"`
	TheaterName string    `json:"theaterName"`
	Date        string    `json:"date"`
	Time        string    `json:"time"`
	Room        int       `json:"room"`
	Features    []string  `json:"features"`
	Audio       string    `json:"audio"`
	SessionID   string    `json:"sessionId"`
	Stats       SeatStats `json:"stats"`
	Error       string    `json:"error,omitempty"`
}

type candidate struct {
	theaterID   int
	theaterName string
	room        cinemark.Room
	session     cinemark.Session
}

func containsInt(list []int, v int) bool {
	if len(list) == 0 {
		return true
	}
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// onlyWantedFeatures vale para a sala quando todos os seus formatos conhecidos
// estão marcados no filtro: desmarcar "3D" tira toda sessão 3D, mesmo que ela
// também seja XD ou Infinity Vision. Sem filtro, tudo passa.
func onlyWantedFeatures(want, have []int) bool {
	if len(want) == 0 {
		return true
	}
	known := 0
	for _, h := range have {
		if _, ok := FeatureNames[h]; !ok {
			continue
		}
		known++
		if !containsInt(want, h) {
			return false
		}
	}
	return known > 0
}

// Select aplica os filtros que não exigem consultar o mapa de assentos.
func Select(days []cinemark.TheaterDay, f Filters) []candidate {
	var out []candidate
	for _, d := range days {
		if !containsInt(f.TheaterIDs, d.TheaterID) {
			continue
		}
		for _, r := range d.Rooms {
			if !onlyWantedFeatures(f.Features, r.Features) || !containsInt(f.Audios, r.Audio) {
				continue
			}
			for _, s := range r.Sessions {
				if s.Expired || len(s.Date) < 16 {
					continue
				}
				date, tm := s.Date[:10], s.Date[11:16]
				if len(f.Dates) > 0 && !contains(f.Dates, date) {
					continue
				}
				if f.TimeFrom != "" && tm < f.TimeFrom {
					continue
				}
				if f.TimeTo != "" && tm > f.TimeTo {
					continue
				}
				out = append(out, candidate{d.TheaterID, d.TheaterName, r, s})
			}
		}
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

type Progress func(done, total int)

// Search busca as sessões do filme, filtra e conta as vagas de cada uma.
func Search(ctx context.Context, c *cinemark.Client, movieID string, cityID int, f Filters, progress Progress) ([]Row, error) {
	days, err := c.Sessions(ctx, movieID, cityID)
	if err != nil {
		return nil, err
	}
	cands := Select(days, f)
	if progress != nil {
		progress(0, len(cands))
	}
	rows := make([]Row, len(cands))
	var done int64
	var wg sync.WaitGroup
	jobs := make(chan int)
	for w := 0; w < 6; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				cd := cands[i]
				row := Row{
					TheaterID: cd.theaterID, TheaterName: cd.theaterName,
					Date: cd.session.Date[:10], Time: cd.session.Date[11:16],
					Room: cd.room.Number, SessionID: cd.session.ID,
					Audio: AudioNames[cd.room.Audio],
				}
				for _, ft := range cd.room.Features {
					if n, ok := FeatureNames[ft]; ok {
						row.Features = append(row.Features, n)
					}
				}
				sm, err := c.SeatMap(ctx, cd.theaterID, cd.session.ID)
				if err != nil {
					row.Error = err.Error()
				} else {
					row.Stats = CountSeats(sm, SeatOptions{GroupSize: f.GroupSize, IncludeSpecial: f.IncludeSpecial, Vertical: f.Vertical, RowFrom: f.RowFrom, RowTo: f.RowTo})
				}
				rows[i] = row
				n := atomic.AddInt64(&done, 1)
				if progress != nil {
					progress(int(n), len(cands))
				}
			}
		}()
	}
	for i := range cands {
		select {
		case jobs <- i:
		case <-ctx.Done():
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	kept := rows[:0]
	for _, r := range rows {
		if r.Error == "" {
			if r.Stats.Free < f.MinFree || (f.OnlyWithGroups && r.Stats.Groups == 0) {
				continue
			}
		}
		kept = append(kept, r)
	}
	sort.SliceStable(kept, func(i, j int) bool {
		a, b := kept[i], kept[j]
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if a.Time != b.Time {
			return a.Time < b.Time
		}
		return strings.Compare(a.TheaterName, b.TheaterName) < 0
	})
	return kept, nil
}
