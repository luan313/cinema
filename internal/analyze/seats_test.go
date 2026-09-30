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
	// Sem tela no mapa, a frente é a maior linha: C(3)=1, D(2)=2, E(1)=3, F(0)=4.
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{mk(3, "C 1"), mk(2, "d 1"), mk(1, "E 1"), mk(0, "F 1")}}
	cases := []struct{ from, to, want int }{
		{0, 0, 4},
		{2, 3, 2},
		{3, 0, 2}, // da 3ª em diante
		{0, 1, 1}, // até a 1ª
		{5, 0, 0},
		{1, 1, 1},
	}
	for _, c := range cases {
		st := CountSeats(sm, SeatOptions{GroupSize: 1, RowFrom: c.from, RowTo: c.to})
		if st.Free != c.want || st.Total != c.want {
			t.Errorf("de %d até %d: %+v, esperado %d", c.from, c.to, st, c.want)
		}
	}
}

func TestRowFilterVerticalStaysInRange(t *testing.T) {
	// D, E e F livres um atrás do outro; só as fileiras 1 e 2 (D, E) entram.
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 3, Col: 1, Name: "D 1", Status: 1, Type: 1, Selectable: true},
		{Row: 2, Col: 1, Name: "E 1", Status: 1, Type: 1, Selectable: true},
		{Row: 1, Col: 1, Name: "F 1", Status: 1, Type: 1, Selectable: true},
		{Row: 4, Type: 7},
	}}
	st := CountSeats(sm, SeatOptions{GroupSize: 3, Vertical: true, RowFrom: 1, RowTo: 2})
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

func TestRowNumbersFollowPositionNotLetters(t *testing.T) {
	// Fileiras da frente para o fundo: AA=1 A=2 B=3 C=4 D=5 E=6 F=7 G=8 H=9 I=10 J=11 K=12 L=13 M=14 N=15.
	sm := flamboyantLike()
	cases := []struct {
		name     string
		from, to int
		want     int
	}{
		{"sem filtro", 0, 0, 15},
		{"da 11ª em diante = J K L M N", 11, 0, 5},
		{"AA (fileira 1) não entra de J em diante", 11, 15, 5},
		{"só a fileira 1 (AA)", 1, 1, 1},
		{"1 a 4: AA A B C", 1, 4, 4},
		{"intervalo invertido é aceito (9 a 3)", 9, 3, 7},
		{"além da maior sala", 20, 0, 0},
		{"até a 20ª = todas", 0, 20, 15},
	}
	for _, c := range cases {
		st := CountSeats(sm, SeatOptions{GroupSize: 1, RowFrom: c.from, RowTo: c.to})
		if st.Free != c.want {
			t.Errorf("%s: %+v, esperado %d", c.name, st, c.want)
		}
	}
}

// Salas menores simplesmente têm menos fileiras: a 11ª não existe numa sala de 3.
func TestRowNumbersSmallRoom(t *testing.T) {
	small := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Name: "C 1", Status: 1, Type: 1, Selectable: true},
		{Row: 2, Name: "B 1", Status: 1, Type: 1, Selectable: true},
		{Row: 3, Name: "A 1", Status: 1, Type: 1, Selectable: true},
		{Row: 4, Type: 7},
	}}
	if st := CountSeats(small, SeatOptions{GroupSize: 1, RowFrom: 2}); st.Free != 2 { // B e C
		t.Fatalf("da 2ª: %+v", st)
	}
	if st := CountSeats(small, SeatOptions{GroupSize: 1, RowFrom: 11}); st.Free != 0 {
		t.Fatalf("da 11ª: %+v", st)
	}
}

func TestRowCount(t *testing.T) {
	small := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Name: "C 1", Status: 1, Type: 1, Selectable: true},
		{Row: 2, Name: "B 1", Status: 1, Type: 1, Selectable: true},
		{Row: 3, Name: "A 1", Status: 1, Type: 1, Selectable: true},
	}}
	if min, max := RowCount([]*cinemark.SeatMap{flamboyantLike(), small}); min != 3 || max != 15 {
		t.Fatalf("min=%d max=%d", min, max)
	}
}
