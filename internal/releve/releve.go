// Package releve fait le Relevé d'un Store : il interroge le Store par
// `plakar at @<store> ls -tags` pour connaître son Dernier snapshot, son
// Snapshot précédent et l'Écart entre les deux.
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
	// Previous est le Snapshot précédent, nil si le Store a moins de deux
	// snapshots.
	Previous *Snapshot
	// Ecart compare Latest à Previous ; nil sans Snapshot précédent.
	Ecart *Ecart
	// Err est l'erreur du Relevé ; le Store est alors Injoignable.
	Err error
}

// Ecart est la variation entre le Dernier snapshot et le Snapshot précédent.
// Ce n'est qu'une indication pour l'œil : il ne change pas l'État du Store.
type Ecart struct {
	// Size est la variation relative de taille : −0.97 pour −97 %.
	Size float64
	// SizeFromZero indique que le Snapshot précédent pesait 0 B, alors que le
	// Dernier snapshot pèse quelque chose : Size n'a alors pas de sens.
	SizeFromZero bool
	// Duration est la durée du Dernier snapshot moins celle du précédent.
	Duration time.Duration
}

func newEcart(latest, previous Snapshot) *Ecart {
	e := &Ecart{Duration: latest.Duration - previous.Duration}
	switch {
	case previous.Size.Bytes > 0:
		e.Size = latest.Size.Bytes/previous.Size.Bytes - 1
	case latest.Size.Bytes > 0:
		e.SizeFromZero = true
	}
	return e
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
		var last []Snapshot
		last, r.Err = parseLast(out, 2)
		if len(last) > 0 {
			r.Latest = &last[0]
		}
		if len(last) > 1 {
			r.Previous = &last[1]
			r.Ecart = newEcart(last[0], last[1])
		}
	}
	return r
}

// parseLast lit les n premiers snapshots de la sortie de `ls -tags`, que
// plakar trie du plus récent au plus ancien. Une sortie vide signifie qu'il
// n'y a aucun snapshot.
func parseLast(out string, n int) ([]Snapshot, error) {
	var last []Snapshot
	sc := bufio.NewScanner(strings.NewReader(out))
	for len(last) < n && sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		s, err := parseLine(sc.Text())
		if err != nil {
			return nil, err
		}
		last = append(last, s)
	}
	return last, sc.Err()
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
