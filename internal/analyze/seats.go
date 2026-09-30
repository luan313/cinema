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
	RowFrom        string // letra da primeira fileira (A = frente, perto da tela); vazio = sem limite
	RowTo          string // letra da última fileira; vazio = sem limite
}

// rowLabel devolve a letra da fileira ("D 4" -> "D"), em maiúsculas.
func rowLabel(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	return strings.ToUpper(f[0])
}

// rowBefore compara letras de fileira: A < B < ... < Z < AA < AB ...
func rowBefore(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// rowInRange diz se a fileira está entre from e to (inclusive). Com limite
// definido, assentos sem letra de fileira ficam de fora.
func rowInRange(label, from, to string) bool {
	if from == "" && to == "" {
		return true
	}
	if label == "" {
		return false
	}
	if from != "" && rowBefore(label, from) {
		return false
	}
	if to != "" && rowBefore(to, label) {
		return false
	}
	return true
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
	free := map[point]bool{}
	for _, e := range sm.Elements {
		if nonSeatTypes[e.Type] || (specialTypes[e.Type] && !includeSpecial) {
			continue
		}
		if !e.Selectable && e.Status == statusFree {
			continue
		}
		if !rowInRange(rowLabel(e.Name), from, to) {
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
