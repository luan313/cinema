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
	st := CountSeats(sm, 2, false)
	if st.Free != 5 || st.Total != 6 || st.BestRun != 3 || st.Groups != 2 {
		t.Fatalf("stats erradas: %+v", st)
	}
	if st := CountSeats(sm, 3, false); st.Groups != 1 {
		t.Fatalf("grupos de 3: %+v", st)
	}
}

func TestSpecialSeats(t *testing.T) {
	sm := &cinemark.SeatMap{Elements: []cinemark.Seat{
		{Row: 1, Col: 1, Status: 1, Type: 5, Selectable: true},
		{Row: 1, Col: 2, Status: 1, Type: 1, Selectable: true},
	}}
	if st := CountSeats(sm, 1, false); st.Free != 1 {
		t.Fatalf("sem especiais: %+v", st)
	}
	if st := CountSeats(sm, 1, true); st.Free != 2 || st.BestRun != 2 {
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
	st := CountSeats(&env.DataResult, 2, false)
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
	st := CountSeats(sm, 2, false)
	if st.Free != 2 || st.Total != 3 || st.Groups != 1 {
		t.Fatalf("namoradeiras: %+v", st)
	}
}
