// Package web sert le tableau de bord.
package web

import (
	"cmp"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"plakar-ui/internal/catalogue"
	"plakar-ui/internal/releve"
)

//go:embed templates/*.html
var templates embed.FS

var page = template.Must(template.ParseFS(templates, "templates/dashboard.html"))

// Handler sert le tableau de bord des Stores du Catalogue. now donne l'heure
// courante et loc le fuseau de l'heure locale affichée.
func Handler(cat *catalogue.Catalogue, now func() time.Time, loc *time.Location, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		rows := make([]row, 0)
		for _, e := range cat.Entries() {
			rows = append(rows, newRow(e, now(), loc))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, rows); err != nil {
			log.Error("rendu du tableau de bord", "err", err)
		}
	})
	return mux
}

// row est une ligne du tableau de bord, prête à afficher.
type row struct {
	Name    string
	Pending bool
	Latest  *snapshot
	// Message remplace le Dernier snapshot quand il n'y en a pas.
	Message string
	// Error est le message d'erreur du dernier Relevé, s'il a échoué.
	Error string
}

// snapshot est le Dernier snapshot d'une ligne, mis en forme, avec l'Écart
// (vide sans Snapshot précédent).
type snapshot struct {
	Date, Age, ID, Size, Duration, Path, Tags string
	SizeEcart, DurationEcart                  string
	// Previous décrit le Snapshot précédent, auquel l'Écart compare.
	Previous string
}

func newRow(e catalogue.Entry, now time.Time, loc *time.Location) row {
	r := row{Name: e.Name}
	switch {
	case e.Releve == nil:
		r.Pending, r.Message = true, "Relevé en attente"
		return r
	case e.Releve.Err != nil:
		r.Message, r.Error = "Relevé en échec", e.Releve.Err.Error()
		return r
	case e.Releve.Latest == nil:
		r.Message = "Aucun snapshot"
		return r
	}
	s := e.Releve.Latest
	r.Latest = &snapshot{
		Date:     s.Date.In(loc).Format("2006-01-02 15:04"),
		Age:      age(now.Sub(s.Date)),
		ID:       s.ID,
		Size:     s.Size.Text,
		Duration: s.Duration.String(),
		Path:     s.Path,
		Tags:     strings.Join(s.Tags, ", "),
	}
	// Le Relevé ne produit un Écart qu'avec un Snapshot précédent.
	if ec, p := e.Releve.Ecart, e.Releve.Previous; ec != nil {
		r.Latest.Previous = fmt.Sprintf("par rapport au Snapshot précédent %s (%s, %s)", p.ID, p.Size.Text, p.Duration)
		r.Latest.SizeEcart = sizeEcart(*ec)
		r.Latest.DurationEcart = signed(cmp.Compare(ec.Duration, 0), ec.Duration.Abs().String())
	}
	return r
}

// sizeEcart écrit l'Écart de taille en pourcentage arrondi : « −97 % ».
func sizeEcart(e releve.Ecart) string {
	if e.SizeFromZero {
		return "nouveau"
	}
	pct := math.Round(e.Size * 100)
	return signed(cmp.Compare(pct, 0), fmt.Sprintf("%.0f\u00a0%%", math.Abs(pct)))
}

// signed préfixe une valeur absolue du signe donné par sign (−1, 0 ou 1) :
// « + », « − », ou rien si elle est nulle.
func signed(sign int, abs string) string {
	switch sign {
	case -1:
		return "−" + abs
	case 1:
		return "+" + abs
	default:
		return abs
	}
}

// age écrit un âge relatif : « il y a 6 h ».
func age(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "à l'instant"
	case d < time.Hour:
		return fmt.Sprintf("il y a %d min", int(d/time.Minute))
	case d < 48*time.Hour:
		return fmt.Sprintf("il y a %d h", int(d/time.Hour))
	default:
		return fmt.Sprintf("il y a %d j", int(d/(24*time.Hour)))
	}
}
