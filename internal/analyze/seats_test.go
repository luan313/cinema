package analyze

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/luan313/cinema/internal/cinemark"
)

func seats(row int, statuses ...int) []cinemark.Seat {
	var out []cinemark.Seat
	for i, s := range statuses {
		out = append(out, cinemark.Seat{Row: row, Col: i + 1, Status: s, Type: 1, Selectable: true})
	}
	return out
}

func TestCountSeats(t *testing.T) {
	sm := &cinemark.SeatMap{}
	// livre livre livre ocupado livre livre  -> corridas 3 e 2
	sm.Elements = append(sm.Elements, seats(1, 1, 1, 1, 3, 1, 1)...)
	// tela e texto não contam
	sm.Elements = append(sm.Elements,
		cinemark.Seat{Row: 1, Col: 0, Type: 9, Status: 1},
		cinemark.Seat{Row: 0, Col: 5, Type: 7, Status: 1})
	st := CountSeats(sm, SeatOptions{GroupSize: 2, IncludeSpecial: false, Vertical: false})
	if st.Free != 5 || st.Total != 6 || st.BestRun != 3 || st.Groups != 2 {
		t.Fatalf("stats erradas: %+v", st)
	}
	if st := CountSeats(sm, SeatOptions{GroupSize: 3, IncludeSpecial: false, Vertical: false}); st.Groups != 1 {
		t.Fatalf("grupos de 3: %+v", st)
	}
}

func TestSpecialSeats(t *testing.T) {
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Col: 1, Status: 1, Type: 5, Selectable: true},
		{Row: 1, Col: 2, Status: 1, Type: 1, Selectable: true},
	}}
	if st := CountSeats(sm, SeatOptions{GroupSize: 1, IncludeSpecial: false, Vertical: false}); st.Free != 1 {
		t.Fatalf("sem especiais: %+v", st)
	}
	if st := CountSeats(sm, SeatOptions{GroupSize: 1, IncludeSpecial: true, Vertical: false}); st.Free != 2 || st.BestRun != 2 {
		t.Fatalf("com especiais: %+v", st)
	}
}

func TestRealSeatMap(t *testing.T) {
	b, err := os.ReadFile("../../testdata/seatmap.json")
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		DataResult cinemark.SeatMap `json:"dataResult"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	st := CountSeats(&env.DataResult, SeatOptions{GroupSize: 2, IncludeSpecial: false, Vertical: false})
	if st.Total == 0 || st.Free > st.Total || st.BestRun < 1 {
		t.Fatalf("mapa real inconsistente: %+v", st)
	}
	t.Logf("%+v", st)
}

func TestLoveseatsCountAsSeats(t *testing.T) {
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Col: 1, Status: 1, Type: 13, Selectable: true},
		{Row: 1, Col: 2, Status: 1, Type: 14, Selectable: true},
		{Row: 1, Col: 3, Status: 3, Type: 13, Selectable: true},
		{Row: 1, Col: 4, Status: 1, Type: 21, Selectable: true},
	}}
	st := CountSeats(sm, SeatOptions{GroupSize: 2, IncludeSpecial: false, Vertical: false})
	if st.Free != 2 || st.Total != 3 || st.Groups != 1 {
		t.Fatalf("namoradeiras: %+v", st)
	}
}

// Duas fileiras: a de trás só tem um assento livre, logo atrás de um livre da frente.
func TestVerticalGroups(t *testing.T) {
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Col: 1, Status: 1, Type: 1, Selectable: true},
		{Row: 1, Col: 2, Status: 3, Type: 1, Selectable: true},
		{Row: 2, Col: 1, Status: 1, Type: 1, Selectable: true},
		{Row: 2, Col: 2, Status: 3, Type: 1, Selectable: true},
	}}
	if st := CountSeats(sm, SeatOptions{GroupSize: 2, IncludeSpecial: false, Vertical: false}); st.Groups != 0 || st.BestRun != 1 {
		t.Fatalf("horizontal: %+v", st)
	}
	if st := CountSeats(sm, SeatOptions{GroupSize: 2, IncludeSpecial: false, Vertical: true}); st.Groups != 1 || st.BestRun != 2 {
		t.Fatalf("com vertical: %+v", st)
	}
}

// Bloco 2x2 livre: cabem dois grupos de 2 e um de 4.
func TestBlock2x2(t *testing.T) {
	var els []cinemark.Seat
	for r := 1; r <= 2; r++ {
		for c := 1; c <= 2; c++ {
			els = append(els, cinemark.Seat{Row: r, Col: c, Status: 1, Type: 1, Selectable: true})
		}
	}
	sm := &cinemark.SeatMap{Elements: els}
	if st := CountSeats(sm, SeatOptions{GroupSize: 2, IncludeSpecial: false, Vertical: true}); st.Groups != 2 || st.BestRun != 4 {
		t.Fatalf("2x2 N=2: %+v", st)
	}
	if st := CountSeats(sm, SeatOptions{GroupSize: 4, IncludeSpecial: false, Vertical: true}); st.Groups != 1 {
		t.Fatalf("2x2 N=4: %+v", st)
	}
	if st := CountSeats(sm, SeatOptions{GroupSize: 3, IncludeSpecial: false, Vertical: false}); st.Groups != 0 {
		t.Fatalf("2x2 N=3 horizontal: %+v", st)
	}
}

// Um corredor (coluna ausente) separa grupos na mesma fileira.
func TestAisleSplits(t *testing.T) {
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Col: 1, Status: 1, Type: 1, Selectable: true},
		{Row: 1, Col: 3, Status: 1, Type: 1, Selectable: true},
	}}
	if st := CountSeats(sm, SeatOptions{GroupSize: 2, IncludeSpecial: false, Vertical: true}); st.Groups != 0 || st.BestRun != 1 {
		t.Fatalf("corredor: %+v", st)
	}
}

func TestRowFilter(t *testing.T) {
	mk := func(row int, name string) cinemark.Seat {
		return cinemark.Seat{Row: row, Col: 1, Name: name, Status: 1, Type: 1, Selectable: true}
	}
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{mk(3, "C 1"), mk(2, "d 1"), mk(1, "E 1"), mk(0, "F 1")}}
	cases := []struct {
		from, to string
		want     int
	}{
		{"", "", 4},
		{"D", "E", 2}, // aceita minúscula no mapa
		{"e", "", 2},  // de E em diante
		{"", "C", 1},  // até C
		{"G", "", 0},
		{"C", "C", 1},
	}
	for _, c := range cases {
		st := CountSeats(sm, SeatOptions{GroupSize: 1, RowFrom: c.from, RowTo: c.to})
		if st.Free != c.want || st.Total != c.want {
			t.Errorf("de %q até %q: %+v, esperado %d", c.from, c.to, st, c.want)
		}
	}
}

func TestRowFilterVerticalStaysInRange(t *testing.T) {
	// D e E livres um atrás do outro; F fica fora do intervalo D-E.
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 3, Col: 1, Name: "D 1", Status: 1, Type: 1, Selectable: true},
		{Row: 2, Col: 1, Name: "E 1", Status: 1, Type: 1, Selectable: true},
		{Row: 1, Col: 1, Name: "F 1", Status: 1, Type: 1, Selectable: true},
	}}
	st := CountSeats(sm, SeatOptions{GroupSize: 3, Vertical: true, RowFrom: "D", RowTo: "E"})
	if st.Groups != 0 || st.BestRun != 2 {
		t.Fatalf("%+v", st)
	}
}

// Reproduz a sala 8 do Flamboyant: a tela fica embaixo (linha 18) e a fileira
// "AA" é a mais próxima dela, na frente da "A".
func flamboyantLike() *cinemark.SeatMap {
	labels := []string{"N", "M", "L", "K", "J", "I", "H", "G", "F", "E", "D", "", "C", "B", "A", "AA"}
	sm := &cinemark.SeatMap{}
	for i, l := range labels {
		if l == "" {
			continue // linha vazia entre as fileiras D e C
		}
		sm.Elements = append(sm.Elements, cinemark.Seat{Row: i + 1, Col: 1, Name: l + " 1", Status: 1, Type: 1, Selectable: true})
	}
	sm.Elements = append(sm.Elements, cinemark.Seat{Row: 18, Col: 0, Name: " ", Type: 7})
	return sm
}

func TestRowFilterUsesPositionNotAlphabet(t *testing.T) {
	sm := flamboyantLike()
	cases := []struct {
		name     string
		from, to string
		want     int
	}{
		{"sem filtro", "", "", 15},
		{"de J em diante (fundo): J K L M N", "J", "", 5},
		{"AA fica na frente e não entra em J+", "J", "N", 5},
		{"de A até C", "A", "C", 3},
		{"AA até B: frente", "AA", "B", 3},
		{"intervalo invertido é aceito", "J", "A", 10}, // A B C D E F G H I J
		{"só AA", "AA", "AA", 1},
		{"até C: AA A B C", "", "C", 4},
	}
	for _, c := range cases {
		st := CountSeats(sm, SeatOptions{GroupSize: 1, RowFrom: c.from, RowTo: c.to})
		if st.Free != c.want {
			t.Errorf("%s: %+v, esperado %d", c.name, st, c.want)
		}
	}
}

func TestRowFilterMissingLetter(t *testing.T) {
	sm := flamboyantLike()
	// remove a fileira I: "de I" passa a valer "de J" (a próxima existente).
	kept := sm.Elements[:0]
	for _, e := range sm.Elements {
		if e.Name != "I 1" {
			kept = append(kept, e)
		}
	}
	sm.Elements = kept
	if st := CountSeats(sm, SeatOptions{GroupSize: 1, RowFrom: "I"}); st.Free != 5 {
		t.Fatalf("de I sem a fileira I: %+v", st)
	}
	if st := CountSeats(sm, SeatOptions{GroupSize: 1, RowFrom: "Z"}); st.Free != 0 {
		t.Fatalf("de Z: %+v", st)
	}
}

func TestRowOrder(t *testing.T) {
	a := flamboyantLike() // AA, A, B, C, D, E ... N
	small := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Name: "C 1", Status: 1, Type: 1, Selectable: true},
		{Row: 2, Name: "B 1", Status: 1, Type: 1, Selectable: true},
		{Row: 3, Name: "A 1", Status: 1, Type: 1, Selectable: true},
		{Row: 4, Type: 7},
	}}
	got := RowOrder([]*cinemark.SeatMap{a, small})
	if got[0] != "AA" || got[1] != "A" || got[len(got)-1] != "N" || len(got) != 15 {
		t.Fatalf("ordem: %v", got)
	}
}
