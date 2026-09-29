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
		{"tudo (menos expiradas)", Filters{}, "abd"},
		{"horário", Filters{TimeFrom: "18:00"}, "bd"},
		{"data", Filters{Dates: []string{"2026-10-01"}}, "d"},
		{"imax", Filters{Features: []int{3}}, "d"},
		{"dublado", Filters{Audios: []int{20}}, "ab"},
		{"outro cinema", Filters{TheaterIDs: []int{2}}, ""},
	}
	for _, c := range cases {
		if got := ids(Select(days, c.f)); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}
