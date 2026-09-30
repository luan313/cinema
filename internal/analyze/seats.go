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
	GroupSize      int    // N do grupo (mínimo 1)
	IncludeSpecial bool   // inclui assentos de acessibilidade/acompanhante
	Vertical       bool   // também vale assento na fileira da frente/de trás
	RowFrom        string // fileira mais próxima da tela do intervalo (ex.: "E"); vazio = sem limite
	RowTo          string // fileira mais distante da tela do intervalo; vazio = sem limite
}

// rowLabel devolve a letra da fileira ("D 4" -> "D"), em maiúsculas.
func rowLabel(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	return strings.ToUpper(f[0])
}

const (
	rankMin = -1 << 30
	rankMax = 1 << 30
)

// rowFilter restringe as fileiras pela posição na sala, e não pela ordem das
// letras: em alguns cinemas a fileira "AA" fica na frente da "A". A posição de
// cada letra é a distância até a tela (0 = a mais próxima).
type rowFilter struct {
	active bool
	rank   map[string]int
	lo, hi int
}

// rowRanks mede a distância de cada letra de fileira até a tela (0 = a mais
// próxima). A tela (tipo 7) marca o lado da frente; sem ela, assume-se que fica
// abaixo da última fileira, como nos mapas do site.
func rowRanks(sm *cinemark.SeatMap) map[string]int {
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
	rank := map[string]int{}
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
		if r, ok := rank[l]; !ok || d < r {
			rank[l] = d
		}
	}
	return rank
}

// RowOrder junta as fileiras de várias salas, da mais perto da tela para a mais
// longe (pela posição média), para montar o seletor de fileiras.
func RowOrder(maps []*cinemark.SeatMap) []string {
	sum, n := map[string]int{}, map[string]int{}
	for _, sm := range maps {
		// Posição ordinal na sala (0 = 1ª fileira a partir da tela), para que o
		// tamanho de cada sala não distorça a média.
		ranks := rowRanks(sm)
		labels := make([]string, 0, len(ranks))
		for l := range ranks {
			labels = append(labels, l)
		}
		sort.Slice(labels, func(i, j int) bool {
			if ranks[labels[i]] != ranks[labels[j]] {
				return ranks[labels[i]] < ranks[labels[j]]
			}
			return labels[i] < labels[j]
		})
		for i, l := range labels {
			sum[l] += i
			n[l]++
		}
	}
	out := make([]string, 0, len(sum))
	for l := range sum {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := float64(sum[out[i]])/float64(n[out[i]]), float64(sum[out[j]])/float64(n[out[j]])
		if a != b {
			return a < b
		}
		return out[i] < out[j]
	})
	return out
}

func newRowFilter(sm *cinemark.SeatMap, from, to string) rowFilter {
	f := rowFilter{lo: rankMin, hi: rankMax}
	if from == "" && to == "" {
		return f
	}
	f.active = true
	f.rank = rowRanks(sm)

	f.lo, f.hi = f.resolveFrom(from), f.resolveTo(to)
	if from != "" && to != "" && f.lo > f.hi { // intervalo escolhido ao contrário
		if lo, hi := f.resolveFrom(to), f.resolveTo(from); lo <= hi {
			f.lo, f.hi = lo, hi
		}
	}
	return f
}

// nearest acha, entre as fileiras de uma letra só, a mais próxima de l
// (a primeira em ordem alfabética >= l se after, senão a última <= l).
func (f rowFilter) nearest(l string, after bool) (int, bool) {
	best, found := "", false
	for c := range f.rank {
		if len(c) != 1 {
			continue
		}
		if after && c >= l && (!found || c < best) || !after && c <= l && (!found || c > best) {
			best, found = c, true
		}
	}
	return f.rank[best], found
}

// resolveFrom devolve a posição inicial do intervalo. Letra ausente na sala:
// vale a próxima letra existente (ex.: sem "I", "de I" começa em "J"); letras
// duplas ausentes (ex.: "AA") não limitam.
func (f rowFilter) resolveFrom(l string) int {
	if l == "" {
		return rankMin
	}
	if r, ok := f.rank[l]; ok {
		return r
	}
	if len(l) > 1 {
		return rankMin
	}
	if r, ok := f.nearest(l, true); ok {
		return r
	}
	return rankMax
}

func (f rowFilter) resolveTo(l string) int {
	if l == "" {
		return rankMax
	}
	if r, ok := f.rank[l]; ok {
		return r
	}
	if len(l) > 1 {
		return rankMax
	}
	if r, ok := f.nearest(l, false); ok {
		return r
	}
	return rankMin
}

func (f rowFilter) allows(label string) bool {
	if !f.active {
		return true
	}
	r, ok := f.rank[label]
	return ok && r >= f.lo && r <= f.hi
}

// CountSeats analisa o mapa. Assentos livres são "juntos" quando vizinhos na
// mesma fileira (colunas consecutivas) ou, com Vertical, também quando estão um
// atrás do outro (mesma coluna, fileiras consecutivas). Corredores aparecem
// como colunas ausentes e, portanto, separam os grupos. Só entram na conta os
// assentos das fileiras entre RowFrom e RowTo.
func CountSeats(sm *cinemark.SeatMap, o SeatOptions) SeatStats {
	groupSize, includeSpecial, vertical := o.GroupSize, o.IncludeSpecial, o.Vertical
	from, to := strings.ToUpper(strings.TrimSpace(o.RowFrom)), strings.ToUpper(strings.TrimSpace(o.RowTo))
	if groupSize < 1 {
		groupSize = 1
	}
	st := SeatStats{GroupSize: groupSize}
	rows := newRowFilter(sm, from, to)
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
