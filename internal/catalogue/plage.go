package catalogue

import (
	"fmt"
	"strconv"
	"strings"
)

// Plage est une plage de ports d'Exploration, bornes comprises.
type Plage struct {
	Debut, Fin int
}

// ParsePlage lit une plage écrite « 9300-9320 ».
func ParsePlage(text string) (Plage, error) {
	a, b, ok := strings.Cut(strings.TrimSpace(text), "-")
	debut, err1 := strconv.Atoi(strings.TrimSpace(a))
	fin, err2 := strconv.Atoi(strings.TrimSpace(b))
	if !ok || err1 != nil || err2 != nil || debut < 1 || fin > 65535 || debut > fin {
		return Plage{}, fmt.Errorf("plage %q : deux ports croissants entre 1 et 65535 sont attendus, par exemple 9300-9320", text)
	}
	return Plage{Debut: debut, Fin: fin}, nil
}

// Contains dit si le port est dans la plage.
func (p Plage) Contains(port int) bool { return port >= p.Debut && port <= p.Fin }

// Ports renvoie les ports de la plage, dans l'ordre.
func (p Plage) Ports() []int {
	ports := make([]int, 0, p.Fin-p.Debut+1)
	for port := p.Debut; port <= p.Fin; port++ {
		ports = append(ports, port)
	}
	return ports
}
