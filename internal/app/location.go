package app

// O app consulta só Goiânia e os dois cinemas abaixo. Os códigos vêm da API do
// Cinemark (/v1/cities e /v1/theaters).
const cityID = 2174 // Goiânia (GO)

var cinemas = map[int]string{
	2133: "Flamboyant",
	2113: "Passeio das Águas",
}

// allowedTheaters devolve os cinemas pedidos que o app atende; vazio (ou nenhum
// válido) devolve os dois.
func allowedTheaters(want []int) []int {
	var out []int
	for _, id := range want {
		if _, ok := cinemas[id]; ok {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		for id := range cinemas {
			out = append(out, id)
		}
	}
	return out
}
