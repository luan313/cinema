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
	BestRun   int `json:"bestRun"`   // maior bloco de assentos livres conectados
	Groups    int `json:"groups"`    // quantos grupos de N conectados cabem (mínimo garantido)
	GroupSize int `json:"groupSize"` // N usado em Groups
}

type point struct{ row, col int }

// CountSeats analisa o mapa. Assentos livres são "juntos" quando vizinhos na
// mesma fileira (colunas consecutivas) ou, com vertical=true, também quando
// estão um atrás do outro (mesma coluna, fileiras consecutivas). Corredores
// aparecem como colunas ausentes e, portanto, separam os grupos.
// includeSpecial inclui assentos de acessibilidade/acompanhante; groupSize é o
// N do grupo (mínimo 1).
func CountSeats(sm *cinemark.SeatMap, groupSize int, includeSpecial, vertical bool) SeatStats {
	if groupSize < 1 {
		groupSize = 1
	}
	st := SeatStats{GroupSize: groupSize}
	free := map[point]bool{}
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
			free[point{e.Row, e.Col}] = true
		}
	}

	// Ordem de varredura estável: de cima para baixo, da esquerda para a direita.
	order := make([]point, 0, len(free))
	for p := range free {
		order = append(order, p)
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].row != order[j].row {
			return order[i].row < order[j].row
		}
		return order[i].col < order[j].col
	})
	neighbors := func(p point) []point {
		n := []point{{p.row, p.col + 1}, {p.row + 1, p.col}, {p.row, p.col - 1}, {p.row - 1, p.col}}
		if !vertical {
			n = []point{{p.row, p.col + 1}, {p.row, p.col - 1}}
		}
		return n
	}

	// Maior bloco conectado.
	seen := map[point]bool{}
	for _, start := range order {
		if seen[start] {
			continue
		}
		size, queue := 0, []point{start}
		seen[start] = true
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			size++
			for _, q := range neighbors(p) {
				if free[q] && !seen[q] {
					seen[q] = true
					queue = append(queue, q)
				}
			}
		}
		if size > st.BestRun {
			st.BestRun = size
		}
	}

	// Grupos de N: recorta blocos conectados de N assentos, começando pelo canto
	// superior esquerdo. É um mínimo garantido (uma heurística gulosa), exato
	// quando só a horizontal conta.
	used := map[point]bool{}
	for _, start := range order {
		if used[start] {
			continue
		}
		group, visited, queue := []point{}, map[point]bool{start: true}, []point{start}
		for len(queue) > 0 && len(group) < groupSize {
			p := queue[0]
			queue = queue[1:]
			group = append(group, p)
			for _, q := range neighbors(p) {
				if free[q] && !used[q] && !visited[q] {
					visited[q] = true
					queue = append(queue, q)
				}
			}
		}
		if len(group) == groupSize {
			for _, p := range group {
				used[p] = true
			}
			st.Groups++
		}
	}
	return st
}
