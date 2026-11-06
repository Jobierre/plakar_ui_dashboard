// Package etat calcule l'État d'un Store et lit les Seuils de retard.
package etat

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"plakar-ui/internal/releve"
)

// Etat est l'État d'un Store.
type Etat int

const (
	// AJour : le Dernier snapshot est plus récent que le Seuil de retard.
	AJour Etat = iota
	// EnRetard : le Dernier snapshot dépasse le Seuil de retard, ou le Store
	// n'a aucun snapshot.
	EnRetard
	// Injoignable : le dernier Relevé a échoué.
	Injoignable
)

func (e Etat) String() string {
	switch e {
	case AJour:
		return "À jour"
	case EnRetard:
		return "En retard"
	default:
		return "Injoignable"
	}
}

// Of calcule l'État d'un Store à l'instant now, à partir de son Dernier
// snapshot (nil s'il n'en a aucun), de son Seuil de retard et de l'erreur de
// son dernier Relevé. Il se calcule à l'affichage : un Store passe En retard
// par le seul passage du temps.
func Of(latest *releve.Snapshot, seuil time.Duration, now time.Time, releveErr error) Etat {
	switch {
	case releveErr != nil:
		return Injoignable
	case latest == nil || now.Sub(latest.Date) > seuil:
		return EnRetard
	default:
		return AJour
	}
}

const day = 24 * time.Hour

// seuilText est un Seuil lisible : des jours facultatifs (« 8d »), suivis
// facultativement d'une durée Go (« 26h », « 1h30m »).
var seuilText = regexp.MustCompile(`^(?:([0-9]+)d)?(.*)$`)

// ParseSeuil lit un Seuil de retard écrit « 26h », « 8d » ou « 1d12h ». Le
// Seuil doit être positif et tenir dans un time.Duration (environ 290 ans).
func ParseSeuil(s string) (time.Duration, error) {
	invalid := fmt.Errorf("Seuil de retard invalide %q : écrivez une durée positive, par exemple 26h, 8d ou 1d12h", s)
	m := seuilText.FindStringSubmatch(strings.TrimSpace(s))
	// Le signe n'est permis nulle part : « 1d-2h » n'est pas un Seuil.
	if m == nil || (m[1] == "" && m[2] == "") || strings.ContainsAny(m[2], "+-") {
		return 0, invalid
	}
	var d time.Duration
	if m[1] != "" {
		days, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || days > math.MaxInt64/int64(day) {
			return 0, invalid
		}
		d = time.Duration(days) * day
	}
	if m[2] != "" {
		rest, err := time.ParseDuration(m[2])
		if err != nil || rest > math.MaxInt64-d {
			return 0, invalid
		}
		d += rest
	}
	if d <= 0 {
		return 0, invalid
	}
	return d, nil
}

// FormatSeuil écrit un Seuil comme ParseSeuil le lit : « 8d » s'il compte des
// jours entiers, sinon « 26h », « 1h30m »…
func FormatSeuil(d time.Duration) string {
	if d%day == 0 {
		return strconv.Itoa(int(d/day)) + "d"
	}
	// time.Duration écrit « 26h0m0s » ou « 1h30m0s » : on retire les zéros
	// de fin.
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = strings.TrimSuffix(s, "0s")
	}
	if strings.HasSuffix(s, "h0m") {
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}
