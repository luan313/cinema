// Package analyze conta vagas em mapas de assentos e aplica filtros às sessões.
package analyze

import (
	"sort"
	"strings"

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

// SeatOptions define o que conta como vaga e como os assentos se agrupam.
type SeatOptions struct {
	GroupSize      int  // N do grupo (mínimo 1)
	IncludeSpecial bool // inclui assentos de acessibilidade/acompanhante
	Vertical       bool // também vale assento na fileira da frente/de trás
	RowFrom        int  // 1ª fileira do intervalo (1 = mais perto da tela); 0 = sem limite
	RowTo          int  // última fileira do intervalo; 0 = até a última da sala
}

// rowLabel devolve a letra da fileira ("D 4" -> "D"), em maiúsculas.
func rowLabel(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	return strings.ToUpper(f[0])
}

// rowOrdinals numera as fileiras da sala a partir da tela: 1 = a mais próxima,
// 2 = a seguinte... A tela (tipo 7) marca o lado da frente; sem ela, assume-se
// que fica abaixo da última fileira, como nos mapas do site. Linhas vazias
// entre blocos não contam, e a letra não importa (algumas salas têm "AA" na
// frente da "A").
func rowOrdinals(sm *cinemark.SeatMap) map[string]int {
	screen, maxRow := 0, 0
	hasScreen := false
	for _, e := range sm.Elements {
		if e.Row > maxRow {
			maxRow = e.Row
		}
		if e.Type == 7 && !hasScreen {
			screen, hasScreen = e.Row, true
		}
	}
	if !hasScreen {
		screen = maxRow + 1
	}
	depth := map[string]int{}
	for _, e := range sm.Elements {
		if e.Type == 7 {
			continue
		}
		l := rowLabel(e.Name)
		if l == "" {
			continue
		}
		d := screen - e.Row
		if d < 0 {
			d = -d
		}
		if cur, ok := depth[l]; !ok || d < cur {
			depth[l] = d
		}
	}
	labels := make([]string, 0, len(depth))
	for l := range depth {
		labels = append(labels, l)
	}
	sort.Slice(labels, func(i, j int) bool {
		if depth[labels[i]] != depth[labels[j]] {
			return depth[labels[i]] < depth[labels[j]]
		}
		return labels[i] < labels[j]
	})
	ord := make(map[string]int, len(labels))
	for i, l := range labels {
		ord[l] = i + 1
	}
	return ord
}

// RowCount devolve a menor e a maior quantidade de fileiras entre as salas.
func RowCount(maps []*cinemark.SeatMap) (min, max int) {
	for i, sm := range maps {
		n := len(rowOrdinals(sm))
		if i == 0 || n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	return min, max
}

// rowFilter limita as fileiras pelo número (1 = mais perto da tela). Zero em
// from/to = sem limite; um intervalo invertido (de 9 até 3) vale como 3 a 9.
type rowFilter struct {
	active   bool
	ord      map[string]int
	from, to int
}

func newRowFilter(sm *cinemark.SeatMap, from, to int) rowFilter {
	if from <= 0 && to <= 0 {
		return rowFilter{}
	}
	if from > 0 && to > 0 && from > to {
		from, to = to, from
	}
	return rowFilter{active: true, ord: rowOrdinals(sm), from: from, to: to}
}

func (f rowFilter) allows(label string) bool {
	if !f.active {
		return true
	}
	o, ok := f.ord[label]
	return ok && (f.from <= 0 || o >= f.from) && (f.to <= 0 || o <= f.to)
}

// CountSeats analisa o mapa. Assentos livres são "juntos" quando vizinhos na
// mesma fileira (colunas consecutivas) ou, com Vertical, também quando estão um
// atrás do outro (mesma coluna, fileiras consecutivas). Corredores aparecem
// como colunas ausentes e, portanto, separam os grupos. Só entram na conta os
// assentos das fileiras entre RowFrom e RowTo.
func CountSeats(sm *cinemark.SeatMap, o SeatOptions) SeatStats {
	groupSize, includeSpecial, vertical := o.GroupSize, o.IncludeSpecial, o.Vertical
	if groupSize < 1 {
		groupSize = 1
	}
	st := SeatStats{GroupSize: groupSize}
	rows := newRowFilter(sm, o.RowFrom, o.RowTo)
	free := map[point]bool{}
	for _, e := range sm.Elements {
		if nonSeatTypes[e.Type] || (specialTypes[e.Type] && !includeSpecial) {
			continue
		}
		if !e.Selectable && e.Status == statusFree {
			continue
		}
		if !rows.allows(rowLabel(e.Name)) {
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
