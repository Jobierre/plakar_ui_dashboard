// Package releve fait le Relevé d'un Store : il interroge le Store par
// `plakar at @<store> ls -tags` pour connaître son Dernier snapshot.
package releve

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"plakar-ui/internal/plakar"
)

// Snapshot est un snapshot tel que listé par `plakar ls -tags`.
type Snapshot struct {
	Date     time.Time
	ID       string // ID court
	Size     Size
	Duration time.Duration
	Path     string // chemin source
	Tags     []string
}

// Size est une taille de snapshot, telle qu'affichée par plakar (« 1.3 GiB »)
// et en octets.
type Size struct {
	Text  string
	Bytes float64
}

// Result est le résultat d'un Relevé.
type Result struct {
	// At est l'heure du Relevé.
	At time.Time
	// Latest est le Dernier snapshot, nil si le Store n'en a aucun.
	Latest *Snapshot
	// Err est l'erreur du Relevé ; le Store est alors Injoignable.
	Err error
}

// Run fait le Relevé du Store. Le Store n'est ouvert que le temps de la
// lecture.
func Run(ctx context.Context, p plakar.Plakar, store string, now func() time.Time) Result {
	out, err := p.Output(ctx, "at", "@"+store, "ls", "-tags")
	r := Result{At: now()}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		r.Err = errors.New("le Store n'a pas répondu dans le délai imparti au Relevé")
	case err != nil:
		r.Err = err
	default:
		r.Latest, r.Err = parseLatest(out)
	}
	return r
}

// parseLatest lit le Dernier snapshot dans la sortie de `ls -tags`, triée du
// plus récent au plus ancien. Une sortie vide signifie qu'il n'y a aucun
// snapshot.
func parseLatest(out string) (*Snapshot, error) {
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		s, err := parseLine(sc.Text())
		if err != nil {
			return nil, err
		}
		return &s, nil
	}
	return nil, sc.Err()
}

// lsLine découpe une ligne de `plakar ls -tags` (plakar v1.1.6) :
//
//	2026-11-04T17:00:03Z   53f9f62e   1.3 GiB       27s /root/docker tags=arr
//
// Le champ tags est absent quand le snapshot n'en a pas, et le chemin source
// peut contenir des espaces.
var lsLine = regexp.MustCompile(`^(\S+)\s+(\S+)\s+([0-9.]+)\s+(\S+)\s+(\S+)\s+(.+?)(?:\s+tags=(\S*))?\s*$`)

// unitBytes donne la valeur en octets des unités binaires de plakar.
var unitBytes = map[string]float64{
	"B": 1, "KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30,
	"TiB": 1 << 40, "PiB": 1 << 50, "EiB": 1 << 60,
}

// parseLine lit une ligne de `plakar ls -tags`.
func parseLine(line string) (Snapshot, error) {
	m := lsLine.FindStringSubmatch(line)
	if m == nil {
		return Snapshot{}, fmt.Errorf("ligne de ls illisible : %q", line)
	}
	date, err := time.Parse(time.RFC3339, m[1])
	if err != nil {
		return Snapshot{}, fmt.Errorf("date illisible dans %q : %w", line, err)
	}
	value, err := strconv.ParseFloat(m[3], 64)
	unit, ok := unitBytes[m[4]]
	if err != nil || !ok {
		return Snapshot{}, fmt.Errorf("taille illisible dans %q", line)
	}
	duration, err := time.ParseDuration(m[5])
	if err != nil {
		return Snapshot{}, fmt.Errorf("durée illisible dans %q : %w", line, err)
	}
	s := Snapshot{
		Date:     date,
		ID:       m[2],
		Size:     Size{Text: m[3] + " " + m[4], Bytes: value * unit},
		Duration: duration,
		Path:     m[6],
	}
	if m[7] != "" {
		s.Tags = strings.Split(m[7], ",")
	}
	return s, nil
}
