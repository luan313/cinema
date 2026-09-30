package analyze

import (
	"testing"

	"github.com/luan313/cinema/internal/cinemark"
)

func TestSelect(t *testing.T) {
	days := []cinemark.TheaterDay{{
		TheaterID: 1, TheaterName: "A",
		Rooms: []cinemark.Room{
			{Number: 1, Features: []int{7}, Audio: 20, Sessions: []cinemark.Session{
				{ID: "a", Date: "2026-09-30T14:00:00"},
				{ID: "b", Date: "2026-09-30T21:30:00"},
				{ID: "c", Date: "2026-09-30T16:00:00", Expired: true},
			}},
			{Number: 2, Features: []int{3}, Audio: 30, Sessions: []cinemark.Session{
				{ID: "d", Date: "2026-10-01T19:00:00"},
			}},
			{Number: 3, Features: []int{6, 10, 2}, Audio: 20, Sessions: []cinemark.Session{
				{ID: "e", Date: "2026-10-01T20:00:00"},
			}},
		},
	}}
	ids := func(c []candidate) (s string) {
		for _, x := range c {
			s += x.session.ID
		}
		return
	}
	cases := []struct {
		name string
		f    Filters
		want string
	}{
		{"tudo (menos expiradas)", Filters{}, "abde"},
		{"horário", Filters{TimeFrom: "18:00"}, "bde"},
		{"data", Filters{Dates: []string{"2026-10-01"}}, "de"},
		{"imax", Filters{Features: []int{3}}, "d"},
		{"3D desmarcado exclui salas 3D+XD+IV", Filters{Features: []int{7, 3, 2, 10}}, "abd"},
		{"todos os formatos da sala marcados", Filters{Features: []int{6, 10, 2}}, "e"},
		{"sem formato conhecido não passa com filtro", Filters{Features: []int{99}}, ""},
		{"dublado", Filters{Audios: []int{20}}, "abe"},
		{"outro cinema", Filters{TheaterIDs: []int{2}}, ""},
	}
	for _, c := range cases {
		if got := ids(Select(days, c.f)); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}
