// Package analyze conta vagas em mapas de assentos e aplica filtros às sessões.
package analyze

import (
	"sort"

	"github.com/luan313/cinema/internal/cinemark"
)

// Códigos vistos na API: status 1=livre, 2=indisponível, 3=ocupado.
const statusFree = 1

// Tipos de elemento vistos na API: 7 tela e 9 texto livre não são assentos;
// 5 cadeirante, 6/18 obeso, 8 acompanhante de cadeirante, 10 mobilidade
// reduzida, 11 acompanhante e 21 VIP cadeirante são especiais. Todo o resto
// (1 normal, 4 VIP, 13/14 namoradeira esquerda/direita...) é assento comum.
var nonSeatTypes = map[int]bool{7: true, 9: true}
var specialTypes = map[int]bool{5: true, 6: true, 8: true, 10: true, 11: true, 18: true, 21: true}

type SeatStats struct {
	Free      int `json:"free"`      // assentos livres
	Total     int `json:"total"`     // assentos da sala (livres + ocupados)
	BestRun   int `json:"bestRun"`   // maior sequência de assentos livres lado a lado
	Groups    int `json:"groups"`    // quantos grupos de N lado a lado cabem
	GroupSize int `json:"groupSize"` // N usado em Groups
}

// CountSeats analisa o mapa. includeSpecial inclui assentos de acessibilidade/
// acompanhante na contagem; groupSize é o N do grupo lado a lado (mínimo 1).
func CountSeats(sm *cinemark.SeatMap, groupSize int, includeSpecial bool) SeatStats {
	if groupSize < 1 {
		groupSize = 1
	}
	st := SeatStats{GroupSize: groupSize}
	rows := map[int][]cinemark.Seat{}
	for _, e := range sm.Elements {
		if nonSeatTypes[e.Type] || (specialTypes[e.Type] && !includeSpecial) {
			continue
		}
		if !e.Selectable && e.Status == statusFree {
			continue
		}
		st.Total++
		if e.Status == statusFree && e.Selectable {
			st.Free++
			rows[e.Row] = append(rows[e.Row], e)
		}
	}
	for _, seats := range rows {
		sort.Slice(seats, func(i, j int) bool { return seats[i].Col < seats[j].Col })
		run := 1
		flush := func() {
			if run > st.BestRun {
				st.BestRun = run
			}
			st.Groups += run / groupSize
		}
		for i := 1; i < len(seats); i++ {
			if seats[i].Col == seats[i-1].Col+1 {
				run++
				continue
			}
			flush()
			run = 1
		}
		flush()
	}
	return st
}
